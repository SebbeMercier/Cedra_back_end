package product

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"

	"cedra_back_end/internal/database"
	"cedra_back_end/internal/services"
)

// =========================
// 🟢 UPLOAD IMAGE PRODUIT
// =========================
func UploadProductImage(c *gin.Context) {
	ctx := context.Background()

	// 1️⃣ Récupérer le fichier
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Fichier manquant"})
		return
	}
	defer file.Close()

	// 2️⃣ Générer un nom unique
	ext := filepath.Ext(header.Filename)
	objectName := fmt.Sprintf("products/%d%s", time.Now().UnixNano(), ext)

	// 3️⃣ Upload vers MinIO
	_, err = database.MinIO.PutObject(
		ctx,
		os.Getenv("MINIO_BUCKET"),
		objectName,
		file,
		header.Size,
		minio.PutObjectOptions{ContentType: header.Header.Get("Content-Type")},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur upload MinIO: " + err.Error()})
		return
	}

	// 4️⃣ Construire l'URL relative
	imageURL := fmt.Sprintf("/uploads/%s", objectName)

	c.JSON(http.StatusOK, gin.H{
		"message":   "✅ Image uploadée avec succès",
		"image_url": imageURL,
	})
}

// =========================
// 🟡 AJOUTER IMAGE À UN PRODUIT
// =========================
func AddImageToProduct(c *gin.Context) {
	var req struct {
		ProductID string `json:"product_id" binding:"required"`
		ImageURL  string `json:"image_url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	productUUID, err := uuid.Parse(req.ProductID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID produit invalide"})
		return
	}

	session, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	// Récupérer les URLs existantes
	var existingURLs []string
	err = session.Query("SELECT image_urls FROM products WHERE product_id = ?", gocql.UUID(productUUID)).Scan(&existingURLs)
	if err != nil && err != gocql.ErrNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur récupération produit"})
		return
	}

	// Ajouter la nouvelle URL
	existingURLs = append(existingURLs, req.ImageURL)

	// Mettre à jour
	err = session.Query("UPDATE products SET image_urls = ? WHERE product_id = ?", existingURLs, gocql.UUID(productUUID)).Exec()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur mise à jour produit"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "✅ Image ajoutée au produit",
		"product_id": req.ProductID,
		"image_url":  req.ImageURL,
	})
}

// =========================
// 🔵 LISTER LES IMAGES D'UN PRODUIT
// =========================
func GetProductImages(c *gin.Context) {
	productID := c.Param("productId")

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

	var imageURLs []string
	err = session.Query("SELECT image_urls FROM products WHERE product_id = ?", gocql.UUID(productUUID)).Scan(&imageURLs)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produit introuvable"})
		return
	}

	// Générer des URLs signées pour MinIO
	ctx := context.Background()
	signedURLs := []string{}

	for _, relativeURL := range imageURLs {
		if relativeURL == "" {
			continue
		}

		// Extraire le chemin après /uploads/
		key := strings.TrimPrefix(relativeURL, "/uploads/")

		// Générer URL signée (valide 24h)
		signed, err := services.GenerateSignedURL(ctx, key, 24*time.Hour)
		if err == nil {
			signedURLs = append(signedURLs, signed)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"product_id": productID,
		"images":     signedURLs,
	})
}

// =========================
// 🔴 SUPPRIMER UNE IMAGE
// =========================
func DeleteProductImage(c *gin.Context) {
	ctx := context.Background()

	var req struct {
		ProductID string `json:"product_id" binding:"required"`
		ImageURL  string `json:"image_url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	productUUID, err := uuid.Parse(req.ProductID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID produit invalide"})
		return
	}

	// Extraire le chemin MinIO
	key := strings.TrimPrefix(req.ImageURL, "/uploads/")

	// Supprimer de MinIO
	err = database.MinIO.RemoveObject(
		ctx,
		os.Getenv("MINIO_BUCKET"),
		key,
		minio.RemoveObjectOptions{},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur suppression MinIO: " + err.Error()})
		return
	}

	// Mettre à jour ScyllaDB
	session, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	var currentURLs []string
	err = session.Query("SELECT image_urls FROM products WHERE product_id = ?", gocql.UUID(productUUID)).Scan(&currentURLs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur récupération produit"})
		return
	}

	// Filtrer l'URL à supprimer
	filteredURLs := []string{}
	for _, url := range currentURLs {
		if url != req.ImageURL {
			filteredURLs = append(filteredURLs, url)
		}
	}

	// Mettre à jour
	err = session.Query("UPDATE products SET image_urls = ? WHERE product_id = ?", filteredURLs, gocql.UUID(productUUID)).Exec()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur mise à jour produit"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "🗑️ Image supprimée avec succès",
		"product_id": req.ProductID,
		"image_url":  req.ImageURL,
	})
}
