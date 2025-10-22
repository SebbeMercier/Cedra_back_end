package product

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
	"cedra_back_end/internal/services"
)

// 🔹 Produit complet avec URLs signées MinIO
func GetProductFull(c *gin.Context) {
	productID := c.Param("id")

	objID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID produit invalide"})
		return
	}

	ctx := context.Background()
	var product models.Product
	err = database.MongoProductsDB.Collection("products").FindOne(ctx, bson.M{"_id": objID}).Decode(&product)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produit introuvable"})
		return
	}

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
