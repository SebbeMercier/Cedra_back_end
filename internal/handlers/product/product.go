package product

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

	// ✅ Validations
	if p.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le champ 'name' est obligatoire"})
		return
	}
	if p.Price <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le prix doit être supérieur à 0"})
		return
	}
	if p.Stock < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le stock ne peut pas être négatif"})
		return
	}
	if p.CategoryID == (gocql.UUID{}) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le champ 'category_id' est obligatoire"})
		return
	}

	// ✅ Connexion ScyllaDB
	session, err := database.GetProductsSession()
	if err != nil {
		log.Printf("❌ Erreur connexion ScyllaDB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	// ✅ Vérifier que la catégorie existe
	var categoryName string
	if err := session.Query(`SELECT name FROM categories WHERE category_id = ?`, p.CategoryID).Scan(&categoryName); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Catégorie introuvable"})
		return
	}

	// ✅ Générer ID et timestamps
	p.ID = gocql.TimeUUID()
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now

	// ✅ Initialiser les slices vides
	if len(p.ImageURLs) == 0 {
		p.ImageURLs = []string{}
	}
	if len(p.Tags) == 0 {
		p.Tags = []string{}
	}

	// ✅ Insérer dans la table principale
	query := `INSERT INTO products (product_id, name, description, price, stock, category_id, image_urls, tags, created_at, updated_at)
              VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	if err := session.Query(query,
		p.ID, p.Name, p.Description, p.Price, p.Stock,
		p.CategoryID, p.ImageURLs, p.Tags,
		p.CreatedAt, p.UpdatedAt,
	).Exec(); err != nil {
		log.Printf("❌ Erreur insertion produit: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur création produit"})
		return
	}

	// ✅ Indexation dans products_by_category (non bloquant)
	go func() {
		if err := session.Query(
			`INSERT INTO products_by_category (category_id, product_id, name, price, stock) VALUES (?, ?, ?, ?, ?)`,
			p.CategoryID, p.ID, p.Name, p.Price, p.Stock,
		).Exec(); err != nil {
			log.Printf("⚠️ Erreur indexation products_by_category: %v", err)
		}
	}()

	// ✅ Indexation Elasticsearch (non bloquant)
	go func() {
		services.IndexProduct(p)
	}()

	// ✅ Invalider le cache Redis
	if database.RedisClient != nil {
		ctx := context.Background()
		database.RedisClient.Del(ctx, "products:all")
	}

	// ✅ Réponse de succès
	c.JSON(http.StatusCreated, gin.H{
		"message":    "✅ Produit créé avec succès",
		"product_id": p.ID.String(),
		"product":    p,
	})
}

func GetAllProducts(c *gin.Context) {
	ctx := context.Background()
	cacheKey := "products:all"

	// 1️⃣ Cache Redis
	if val, err := database.RedisClient.Get(ctx, cacheKey).Result(); err == nil && val != "" {
		var cached []models.Product
		if err := json.Unmarshal([]byte(val), &cached); err == nil {
			// ✅ Générer les URLs signées pour chaque produit
			for i := range cached {
				signed := []string{}
				for _, url := range cached[i].ImageURLs {
					if url != "" {
						key := strings.TrimPrefix(url, "/uploads/")
						signedURL, err := services.GenerateSignedURL(ctx, key, 24*time.Hour)
						if err == nil {
							signed = append(signed, signedURL)
						}
					}
				}
				cached[i].ImageURLs = signed
			}
			c.JSON(http.StatusOK, cached)
			return
		}
	}

	session, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	iter := session.Query(
		`SELECT product_id, name, description, price, stock, category_id, image_urls, tags, created_at, updated_at  FROM products`,
	).Iter()

	var products []models.Product
	var p models.Product

	for iter.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.CategoryID, &p.ImageURLs, &p.Tags, &p.CreatedAt, &p.UpdatedAt) {
		// ✅ Générer les URLs signées MinIO
		signed := []string{}
		for _, url := range p.ImageURLs {
			if url != "" {
				key := strings.TrimPrefix(url, "/uploads/")
				signedURL, err := services.GenerateSignedURL(ctx, key, 24*time.Hour)
				if err == nil {
					signed = append(signed, signedURL)
				}
			}
		}
		p.ImageURLs = signed
		products = append(products, p)
		p = models.Product{}
	}

	if err := iter.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lecture produits: " + err.Error()})
		return
	}

	// 3️⃣ Mise en cache (sans les URLs signées)
	if data, err := json.Marshal(products); err == nil {
		database.RedisClient.Set(ctx, cacheKey, data, 30*time.Minute)
	}

	c.JSON(http.StatusOK, products)
}

func SearchProducts(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'q' is required"})
		return
	}

	// ✅ Validation : bloquer caractères dangereux
	if strings.ContainsAny(query, ";'\"\\") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid characters in search query"})
		return
	}

	ctx := context.Background()

	// 1️⃣ Recherche Elasticsearch (prioritaire)
	results, err := services.SearchProducts(query)
	if err == nil && len(results) > 0 {
		// ✅ Générer URLs signées pour Elasticsearch
		for i := range results {
			if urls, ok := results[i]["image_urls"].([]interface{}); ok {
				signed := []string{}
				for _, u := range urls {
					if str, ok := u.(string); ok && str != "" {
						key := strings.TrimPrefix(str, "/uploads/")
						signedURL, err := services.GenerateSignedURL(ctx, key, 24*time.Hour)
						if err == nil {
							signed = append(signed, signedURL)
						}
					}
				}
				results[i]["image_urls"] = signed
			}
		}

		// ✅ Format JSON standardisé
		c.JSON(http.StatusOK, gin.H{
			"products": results,
			"count":    len(results),
			"source":   "elasticsearch",
		})
		return
	}

	// 2️⃣ Fallback ScyllaDB
	session, err := database.GetProductsSession()
	if err != nil {
		log.Printf("❌ Erreur connexion ScyllaDB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	iter := session.Query(
		`SELECT product_id, name, description, price, stock, category_id, image_urls, tags, created_at, updated_at 
         FROM products`,
	).Iter()

	var products []models.Product
	var p models.Product

	for iter.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.CategoryID, &p.ImageURLs, &p.Tags, &p.CreatedAt, &p.UpdatedAt) {
		// ✅ Filtre en mémoire (Description est maintenant un string, pas *string)
		if containsIgnoreCase(p.Name, query) ||
			containsIgnoreCase(p.Description, query) ||
			containsTagsIgnoreCase(p.Tags, query) {

			// ✅ Générer URLs signées
			signed := []string{}
			for _, url := range p.ImageURLs {
				if url != "" {
					key := strings.TrimPrefix(url, "/uploads/")
					signedURL, err := services.GenerateSignedURL(ctx, key, 24*time.Hour)
					if err == nil {
						signed = append(signed, signedURL)
					}
				}
			}
			p.ImageURLs = signed
			products = append(products, p)
		}
		p = models.Product{} // Reset
	}

	if err := iter.Close(); err != nil {
		log.Printf("❌ Erreur ScyllaDB iter.Close(): %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search error"})
		return
	}

	// ✅ Format JSON standardisé
	c.JSON(http.StatusOK, gin.H{
		"products": products,
		"count":    len(products),
		"source":   "scylladb",
	})
}

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

	ctx := context.Background()
	cacheKey := fmt.Sprintf("products:category:%s", categoryID)

	// 1️⃣ Cache Redis
	if val, err := database.RedisClient.Get(ctx, cacheKey).Result(); err == nil && val != "" {
		var cached []models.Product
		if err := json.Unmarshal([]byte(val), &cached); err == nil {
			// ✅ Générer URLs signées
			for i := range cached {
				signed := []string{}
				for _, url := range cached[i].ImageURLs {
					if url != "" {
						key := strings.TrimPrefix(url, "/uploads/")
						signedURL, err := services.GenerateSignedURL(ctx, key, 24*time.Hour)
						if err == nil {
							signed = append(signed, signedURL)
						}
					}
				}
				cached[i].ImageURLs = signed
			}
			c.JSON(http.StatusOK, cached)
			return
		}
	}

	// 2️⃣ ScyllaDB - table products_by_category (optimisé)
	session, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	iter := session.Query(
		`SELECT product_id, name, price, stock FROM products_by_category WHERE category_id = ?`,
		catUUID,
	).Iter()

	var productIDs []gocql.UUID
	var basicProducts []models.Product
	var p models.Product

	for iter.Scan(&p.ID, &p.Name, &p.Price, &p.Stock) {
		productIDs = append(productIDs, p.ID)
		basicProducts = append(basicProducts, p)
		p = models.Product{}
	}

	if err := iter.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lecture produits: " + err.Error()})
		return
	}

	// 3️⃣ Enrichir avec les détails complets (description, images, etc.)
	var products []models.Product
	for _, basicProd := range basicProducts {
		var fullProd models.Product
		err := session.Query(
			`SELECT product_id, name, description, price, stock, category_id, image_urls, tags, created_at, updated_at 
            FROM products WHERE product_id = ?`,
			basicProd.ID,
		).Scan(
			&fullProd.ID,
			&fullProd.Name,
			&fullProd.Description,
			&fullProd.Price,
			&fullProd.Stock,
			&fullProd.CategoryID, // ✅ Virgule ajoutée
			&fullProd.ImageURLs,
			&fullProd.Tags,
			&fullProd.CreatedAt,
			&fullProd.UpdatedAt,
		)

		if err == nil {
			// ✅ Générer URLs signées
			signed := []string{}
			for _, url := range fullProd.ImageURLs {
				if url != "" {
					key := strings.TrimPrefix(url, "/uploads/")
					signedURL, err := services.GenerateSignedURL(ctx, key, 24*time.Hour)
					if err == nil {
						signed = append(signed, signedURL)
					}
				}
			}
			fullProd.ImageURLs = signed
			products = append(products, fullProd)
		}
	}

	// 4️⃣ Mise en cache
	if data, err := json.Marshal(products); err == nil {
		database.RedisClient.Set(ctx, cacheKey, data, 30*time.Minute)
	}

	c.JSON(http.StatusOK, products)
}

func GetBestSellers(c *gin.Context) {
	ctx := context.Background()
	cacheKey := "products:bestsellers"

	if val, err := database.RedisClient.Get(ctx, cacheKey).Result(); err == nil && val != "" {
		var cached []models.Product
		if err := json.Unmarshal([]byte(val), &cached); err == nil {
			// ✅ Générer URLs signées
			for i := range cached {
				signed := []string{}
				for _, url := range cached[i].ImageURLs {
					if url != "" {
						key := strings.TrimPrefix(url, "/uploads/")
						signedURL, err := services.GenerateSignedURL(ctx, key, 24*time.Hour)
						if err == nil {
							signed = append(signed, signedURL)
						}
					}
				}
				cached[i].ImageURLs = signed
			}
			c.JSON(http.StatusOK, cached)
			return
		}
	}

	ordersSession, err := database.GetOrdersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion orders"})
		return
	}

	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	iter := ordersSession.Query(
		`SELECT items FROM orders WHERE created_at >= ? ALLOW FILTERING`,
		thirtyDaysAgo,
	).Iter()

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

	type productSale struct {
		ProductID string
		Quantity  int
	}

	var sales []productSale
	for pid, qty := range productSales {
		sales = append(sales, productSale{ProductID: pid, Quantity: qty})
	}

	limit := 10
	if len(sales) < limit {
		limit = len(sales)
	}

	productsSession, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion products"})
		return
	}

	var products []models.Product
	for i := 0; i < limit && i < len(sales); i++ {
		sale := sales[i]
		productUUID, err := gocql.ParseUUID(sale.ProductID)
		if err != nil {
			continue
		}

		var p models.Product
		err = productsSession.Query(
			`SELECT product_id, name, description, price, stock, category_id, image_urls, tags, created_at, updated_at 
            FROM products WHERE product_id = ?`,
			productUUID,
		).Scan(
			&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock,
			&p.CategoryID, &p.ImageURLs, &p.Tags,
			&p.CreatedAt, &p.UpdatedAt,
		)

		if err == nil {
			// ✅ Générer URLs signées
			signed := []string{}
			for _, url := range p.ImageURLs {
				if url != "" {
					key := strings.TrimPrefix(url, "/uploads/")
					signedURL, err := services.GenerateSignedURL(ctx, key, 24*time.Hour)
					if err == nil {
						signed = append(signed, signedURL)
					}
				}
			}
			p.ImageURLs = signed
			products = append(products, p)
		}
	}

	// 5️⃣ Cache pour 1 heure (calcul coûteux)
	if data, err := json.Marshal(products); err == nil {
		database.RedisClient.Set(ctx, cacheKey, data, 1*time.Hour)
	}

	c.JSON(http.StatusOK, products)
}
