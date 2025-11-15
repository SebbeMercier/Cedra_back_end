package product

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"

	"cedra_back_end/internal/database"
	services "cedra_back_end/internal/services"
)

// =========================
// 🟢 UPLOAD IMAGE PRODUIT
// =========================
func UploadProductImage(c *gin.Context) {
	ctx := context.Background()

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Form-data invalide"})
		return
	}

	files := form.File["files"] // clé = "files[]" côté front
	productID := form.Value["product_id"]
	if len(productID) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Champ 'product_id' manquant"})
		return
	}

	productUUID, err := uuid.Parse(productID[0])
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID produit invalide"})
		return
	}

	uploadedURLs := []string{}

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			continue
		}
		defer file.Close()

		objectName := fmt.Sprintf("products/%d-%s", time.Now().UnixNano(), fileHeader.Filename)
		_, err = database.MinIO.PutObject(
			ctx,
			os.Getenv("MINIO_BUCKET"),
			objectName,
			file,
			fileHeader.Size,
			minio.PutObjectOptions{ContentType: fileHeader.Header.Get("Content-Type")},
		)
		if err != nil {
			continue
		}

		imageURL := fmt.Sprintf("http://%s/%s/%s",
			os.Getenv("MINIO_ENDPOINT"),
			os.Getenv("MINIO_BUCKET"),
			objectName,
		)
		uploadedURLs = append(uploadedURLs, imageURL)
	}

	if len(uploadedURLs) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Aucune image uploadée"})
		return
	}

	// 🔹 Ajout en masse dans ScyllaDB
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

	// Fusionner les URLs existantes avec les nouvelles
	allURLs := append(existingURLs, uploadedURLs...)

	// Mettre à jour le produit
	err = session.Query("UPDATE products SET image_urls = ? WHERE product_id = ?", allURLs, gocql.UUID(productUUID)).Exec()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur mise à jour ScyllaDB"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "✅ Images uploadées avec succès",
		"uploaded":   uploadedURLs,
		"product_id": productID[0],
		"count":      len(uploadedURLs),
	})
}

// =========================
// 🟡 LISTER LES IMAGES D’UN PRODUIT
// =========================
func GetProductImages(c *gin.Context) {
	ctx := context.Background()
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

	signedURLs := []string{}
	for _, rawURL := range imageURLs {
		if len(rawURL) == 0 {
			continue
		}

		// Extrait la partie après le bucket
		const prefix = "http://192.168.1.130:9000/cedra-images/"
		key := rawURL
		if len(rawURL) > len(prefix) {
			key = rawURL[len(prefix):]
		}

		// ✅ Appel correct à GenerateSignedURL (nouvelle signature)
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
	imageID := c.Param("id") // ex: products/172928721234-tournevis.jpg

	// Supprime l'objet de MinIO
	err := database.MinIO.RemoveObject(
		ctx,
		os.Getenv("MINIO_BUCKET"),
		imageID,
		minio.RemoveObjectOptions{},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur suppression MinIO"})
		return
	}

	// Supprime aussi l'URL du champ image_urls dans ScyllaDB
	imageURL := fmt.Sprintf("http://%s/%s/%s",
		os.Getenv("MINIO_ENDPOINT"),
		os.Getenv("MINIO_BUCKET"),
		imageID,
	)

	session, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	// Récupérer tous les produits qui contiennent cette URL (peut nécessiter ALLOW FILTERING)
	// Pour l'instant, on va chercher tous les produits et filtrer en mémoire
	// Note: Pour de meilleures performances, on pourrait créer une table d'index
	iter := session.Query("SELECT product_id, image_urls FROM products").Iter()
	var (
		prodID   gocql.UUID
		prodURLs []string
	)
	productsToUpdate := []gocql.UUID{}
	for iter.Scan(&prodID, &prodURLs) {
		// Vérifier si l'URL est dans la liste
		for _, url := range prodURLs {
			if url == imageURL {
				productsToUpdate = append(productsToUpdate, prodID)
				break
			}
		}
	}
	iter.Close()

	// Mettre à jour chaque produit pour retirer l'URL
	for _, prodID := range productsToUpdate {
		// Récupérer les URLs actuelles
		var currentURLs []string
		err = session.Query("SELECT image_urls FROM products WHERE product_id = ?", prodID).Scan(&currentURLs)
		if err != nil {
			continue
		}

		// Filtrer l'URL à supprimer
		filteredURLs := []string{}
		for _, url := range currentURLs {
			if url != imageURL {
				filteredURLs = append(filteredURLs, url)
			}
		}

		// Mettre à jour
		session.Query("UPDATE products SET image_urls = ? WHERE product_id = ?", filteredURLs, prodID).Exec()
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "🗑️ Image supprimée avec succès",
		"image_id": imageID,
	})
}
