package product

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"

	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
	"cedra_back_end/internal/services"
)

func CreateProduct(c *gin.Context) {
	var p models.Product

	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Vérifie que CategoryID est fourni
	if p.CategoryID == (gocql.UUID{}) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le champ 'category_id' est obligatoire"})
		return
	}

	// ✅ Vérifie la catégorie dans ScyllaDB
	session, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	var categoryName string
	if err := session.Query(`SELECT name FROM categories WHERE category_id = ?`, p.CategoryID).Scan(&categoryName); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Catégorie introuvable"})
		return
	}

	// ✅ Génère automatiquement l'URL MinIO si tu sais où l'image est stockée
	if len(p.ImageURLs) == 0 || p.ImageURLs[0] == "" {
		imageURL := fmt.Sprintf("http://%s/%s/products/%s.jpg",
			os.Getenv("MINIO_ENDPOINT"),
			os.Getenv("MINIO_BUCKET"),
			p.Name,
		)
		p.ImageURLs = []string{imageURL}
	}

	// ✅ Génère un nouvel UUID pour le produit
	p.ID = gocql.TimeUUID()
	now := time.Now()
	p.CreatedAt = &now
	p.UpdatedAt = &now

	// ✅ Sauvegarde dans ScyllaDB
	query := `INSERT INTO products (product_id, name, description, price, stock, category_id, company_id, image_urls, tags, created_at, updated_at) 
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	if err := session.Query(query, p.ID, p.Name, p.Description, p.Price, p.Stock, p.CategoryID, p.CompanyID, p.ImageURLs, p.Tags, p.CreatedAt, p.UpdatedAt).Exec(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur création produit: " + err.Error()})
		return
	}

	// ✅ Indexe aussi dans products_by_category pour les requêtes par catégorie
	if err := session.Query(`INSERT INTO products_by_category (category_id, product_id, name, price, stock) VALUES (?, ?, ?, ?, ?)`,
		p.CategoryID, p.ID, p.Name, p.Price, p.Stock).Exec(); err != nil {
		// Log l'erreur mais ne bloque pas la création
		fmt.Printf("⚠️ Erreur indexation products_by_category: %v\n", err)
	}

	// 🔄 Indexation Elasticsearch
	go services.IndexProduct(p)

	c.JSON(http.StatusOK, p)
}

func GetAllProducts(c *gin.Context) {
	ctx := context.Background()
	cacheKey := "products:all"

	// ✅ Vérifie le cache Redis
	if val, err := database.RedisClient.Get(ctx, cacheKey).Result(); err == nil && val != "" {
		var cached []models.Product
		if err := json.Unmarshal([]byte(val), &cached); err == nil {
			c.JSON(http.StatusOK, cached)
			return
		}
	}

	// ✅ Récupère depuis ScyllaDB
	session, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	iter := session.Query(`SELECT product_id, name, description, price, stock, category_id, company_id, image_urls, tags, created_at, updated_at FROM products`).Iter()

	var products []models.Product
	var p models.Product

	for iter.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.CategoryID, &p.CompanyID, &p.ImageURLs, &p.Tags, &p.CreatedAt, &p.UpdatedAt) {
		products = append(products, p)
		p = models.Product{} // Reset pour la prochaine itération
	}

	if err := iter.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lecture produits: " + err.Error()})
		return
	}

	// ✅ Met en cache
	if data, err := json.Marshal(products); err == nil {
		database.RedisClient.Set(ctx, cacheKey, data, time.Hour)
	}

	c.JSON(http.StatusOK, products)
}

func SearchProducts(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "paramètre 'q' manquant"})
		return
	}

	// 🔎 1️⃣ Recherche dans Elasticsearch (prioritaire)
	results, err := services.SearchProducts(query)
	if err == nil && len(results) > 0 {
		// ✅ Génère les URLs signées MinIO pour chaque produit
		for i := range results {
			if urls, ok := results[i]["image_urls"].([]interface{}); ok {
				signed := []string{}
				for _, u := range urls {
					if str, ok := u.(string); ok && str != "" {
						signedURL, err := services.GenerateSignedURL(context.Background(), str, 24*time.Hour)
						if err == nil {
							signed = append(signed, signedURL)
						}
					}
				}
				results[i]["image_urls"] = signed
			}
		}
		c.JSON(http.StatusOK, results)
		return
	}

	// 🔁 2️⃣ Fallback ScyllaDB si ES vide (scan complet - non optimal pour production)
	ctx := context.Background()
	session, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	// Note: ScyllaDB ne supporte pas les recherches LIKE/regex natives
	// Cette approche charge tous les produits et filtre en mémoire
	// Pour la production, utilisez Elasticsearch ou créez des index secondaires
	iter := session.Query(`SELECT product_id, name, description, price, stock, category_id, company_id, image_urls, tags, created_at, updated_at FROM products`).Iter()

	var products []models.Product
	var p models.Product

	for iter.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.CategoryID, &p.CompanyID, &p.ImageURLs, &p.Tags, &p.CreatedAt, &p.UpdatedAt) {
		// Filtre en mémoire (non optimal)
		if containsIgnoreCase(p.Name, query) || containsIgnoreCase(p.Description, query) || containsTagsIgnoreCase(p.Tags, query) {
			// ✅ Génère les URLs signées MinIO
			signed := []string{}
			for _, url := range p.ImageURLs {
				signedURL, err := services.GenerateSignedURL(ctx, url, 24*time.Hour)
				if err == nil {
					signed = append(signed, signedURL)
				}
			}
			p.ImageURLs = signed
			products = append(products, p)
		}
		p = models.Product{}
	}

	if err := iter.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur recherche: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, products)
}

// Helper pour recherche insensible à la casse
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func containsTagsIgnoreCase(tags []string, query string) bool {
	for _, tag := range tags {
		if containsIgnoreCase(tag, query) {
			return true
		}
	}
	return false
}

func GetProductsByCategory(c *gin.Context) {
	categoryID := c.Param("id")

	// Parse UUID
	catUUID, err := gocql.ParseUUID(categoryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de catégorie invalide"})
		return
	}

	// ✅ Utilise la table products_by_category pour une requête optimisée
	session, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	iter := session.Query(`SELECT product_id, name, price, stock FROM products_by_category WHERE category_id = ?`, catUUID).Iter()

	var products []models.Product
	var p models.Product

	for iter.Scan(&p.ID, &p.Name, &p.Price, &p.Stock) {
		products = append(products, p)
		p = models.Product{}
	}

	if err := iter.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lecture produits: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, products)
}

func GetBestSellers(c *gin.Context) {
	// Note: ScyllaDB ne supporte pas les agrégations complexes comme MongoDB
	// Pour les best-sellers, vous devriez:
	// 1. Utiliser une table matérialisée mise à jour périodiquement
	// 2. Calculer les stats avec un job batch (Spark, etc.)
	// 3. Stocker les résultats dans Redis ou une table dédiée

	// Pour l'instant, retournons une implémentation simplifiée
	// qui récupère les commandes récentes et calcule en mémoire

	ordersSession, err := database.GetOrdersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	// Récupère les commandes des 30 derniers jours
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	// Note: Cette requête nécessite un index sur created_at ou un scan complet
	// Pour la production, créez une table best_sellers mise à jour par un job
	iter := ordersSession.Query(`SELECT items FROM orders WHERE created_at >= ? ALLOW FILTERING`, thirtyDaysAgo).Iter()

	// Map pour compter les ventes par produit
	productSales := make(map[string]int)
	var itemsJSON string

	for iter.Scan(&itemsJSON) {
		var items []models.OrderItem
		if err := json.Unmarshal([]byte(itemsJSON), &items); err == nil {
			for _, item := range items {
				productSales[item.ProductID] += item.Quantity
			}
		}
	}

	if err := iter.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lecture commandes: " + err.Error()})
		return
	}

	// Trie les produits par nombre de ventes (implémentation simple)
	type productSale struct {
		ProductID string
		Quantity  int
	}

	var sales []productSale
	for pid, qty := range productSales {
		sales = append(sales, productSale{ProductID: pid, Quantity: qty})
	}

	// Limite aux 10 premiers (tri simple pour l'exemple)
	limit := 10
	if len(sales) > limit {
		sales = sales[:limit]
	}

	// Récupère les détails des produits
	productsSession, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	var products []models.Product
	for _, sale := range sales {
		productUUID, err := gocql.ParseUUID(sale.ProductID)
		if err != nil {
			continue
		}

		var p models.Product
		if err := productsSession.Query(`SELECT product_id, name, description, price, stock, category_id, image_urls FROM products WHERE product_id = ?`,
			productUUID).Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.CategoryID, &p.ImageURLs); err == nil {
			products = append(products, p)
		}
	}

	c.JSON(http.StatusOK, products)
}
