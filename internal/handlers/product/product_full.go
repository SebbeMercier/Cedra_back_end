package product

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/google/uuid"

	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
	"cedra_back_end/internal/services"
)

func GetProductFull(c *gin.Context) {
	productID := c.Param("id")

	productUUID, err := uuid.Parse(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID produit invalide"})
		return
	}

	ctx := context.Background()
	cacheKey := fmt.Sprintf("product:full:%s", productID)

	if val, err := database.RedisClient.Get(ctx, cacheKey).Result(); err == nil && val != "" {
		var cached models.Product
		if err := json.Unmarshal([]byte(val), &cached); err == nil {
			// ✅ Générer URLs signées (même pour le cache)
			signed := []string{}
			for _, url := range cached.ImageURLs {
				if url != "" {
					key := extractMinIOKey(url)
					signedURL, err := services.GenerateSignedURL(ctx, key, 24*time.Hour)
					if err == nil {
						signed = append(signed, signedURL)
					}
				}
			}
			cached.ImageURLs = signed
			c.JSON(http.StatusOK, cached)
			return
		}
	}

	session, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	var product models.Product

	err = session.Query(
		`SELECT product_id, name, description, price, stock, category_id, company_id, image_urls, tags, created_at, updated_at 
        FROM products WHERE product_id = ?`,
		gocql.UUID(productUUID),
	).Scan(
		&product.ID,
		&product.Name,
		&product.Description,
		&product.Price,
		&product.Stock,
		&product.CategoryID,
		&product.CompanyID,
		&product.ImageURLs,
		&product.Tags,
		&product.CreatedAt,
		&product.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produit introuvable"})
		return
	}

	signedURLs := []string{}
	for _, url := range product.ImageURLs {
		if url != "" {
			key := extractMinIOKey(url)
			signedURL, err := services.GenerateSignedURL(ctx, key, 24*time.Hour)
			if err == nil {
				signedURLs = append(signedURLs, signedURL)
			}
		}
	}
	product.ImageURLs = signedURLs

	productToCache := product
	productToCache.ImageURLs = extractOriginalURLs(product.ImageURLs)

	if data, err := json.Marshal(productToCache); err == nil {
		database.RedisClient.Set(ctx, cacheKey, data, 15*time.Minute)
	}

	c.JSON(http.StatusOK, product)
}

func extractMinIOKey(url string) string {
	// Supprimer le préfixe MinIO si présent
	// Ex: "http://localhost:9000/cedra-images/products/xxx.jpg" -> "products/xxx.jpg"

	if idx := strings.Index(url, "/cedra-images/"); idx != -1 {
		return url[idx+len("/cedra-images/"):]
	}

	// Si c'est déjà un chemin relatif
	return strings.TrimPrefix(url, "/uploads/")
}

func extractOriginalURLs(signedURLs []string) []string {
	var original []string
	for _, signedURL := range signedURLs {
		// Extraire le chemin de l'URL signée
		// Ex: "http://localhost:9000/cedra-images/products/xxx.jpg?X-Amz-..."
		//     -> "products/xxx.jpg"
		if idx := strings.Index(signedURL, "?"); idx != -1 {
			signedURL = signedURL[:idx]
		}
		key := extractMinIOKey(signedURL)
		original = append(original, "/uploads/"+key)
	}
	return original
}
