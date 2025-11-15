package product

import (
	"context"
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

// 🔹 Produit complet avec URLs signées MinIO
func GetProductFull(c *gin.Context) {
	productID := c.Param("id")

	productUUID, err := uuid.Parse(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID produit invalide"})
		return
	}

	session, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	ctx := context.Background()
	var product models.Product
	var (
		name, description    string
		price                float64
		stock                int
		categoryID           gocql.UUID
		imageURLs            []string
		tags                 []string
		companyID            gocql.UUID
		createdAt, updatedAt *time.Time
	)

	err = session.Query(`SELECT product_id, name, description, price, stock, category_id, company_id, image_urls, tags, created_at, updated_at 
	                     FROM products WHERE product_id = ?`, gocql.UUID(productUUID)).Scan(
		&product.ID, &name, &description, &price, &stock, &categoryID, &companyID, &imageURLs, &tags, &createdAt, &updatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produit introuvable"})
		return
	}

	product.Name = name
	product.Description = description
	product.Price = price
	product.Stock = stock
	product.CategoryID = categoryID
	product.CompanyID = companyID
	product.ImageURLs = imageURLs
	product.Tags = tags
	product.CreatedAt = createdAt
	product.UpdatedAt = updatedAt

	// 🔹 Génère des URLs signées (valables 24h)
	signedURLs := []string{}
	ctx = context.Background() // ✅ Ajout du contexte

	for _, img := range product.ImageURLs {
		if img == "" {
			continue
		}

		// Extraire juste le chemin à partir du bucket
		path := img
		if idx := strings.Index(img, "/cedra-images/"); idx != -1 {
			path = img[idx+len("/cedra-images/"):]
		}

		// ✅ Appel corrigé avec les bons arguments
		signed, err := services.GenerateSignedURL(ctx, path, 24*time.Hour)
		if err == nil {
			signedURLs = append(signedURLs, signed)
		}
	}

	product.ImageURLs = signedURLs
	c.JSON(http.StatusOK, product)
}
