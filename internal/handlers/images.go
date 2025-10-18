package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"cedra_back_end/internal/database"
	services "cedra_back_end/internal/services"
)

//
// =========================
// 🟢 UPLOAD IMAGE PRODUIT
// =========================
//
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

	objID, err := primitive.ObjectIDFromHex(productID[0])
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
		_, err = database.MinioClient.PutObject(
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

	// 🔹 Ajout en masse dans MongoDB
	collection := database.MongoProductsDB.Collection("products")
	_, err = collection.UpdateOne(
		ctx,
		bson.M{"_id": objID},
		bson.M{"$push": bson.M{"image_urls": bson.M{"$each": uploadedURLs}}},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur mise à jour MongoDB"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "✅ Images uploadées avec succès",
		"uploaded":    uploadedURLs,
		"product_id":  productID[0],
		"count":       len(uploadedURLs),
	})
}

//
// =========================
// 🟡 LISTER LES IMAGES D’UN PRODUIT
// =========================
//
func GetProductImages(c *gin.Context) {
	ctx := context.Background()
	productID := c.Param("productId")

	objID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID produit invalide"})
		return
	}

	collection := database.MongoProductsDB.Collection("products")
	var product struct {
		ImageURLs []string `bson:"image_urls"`
	}
	err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&product)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produit introuvable"})
		return
	}

	signedURLs := []string{}
	for _, rawURL := range product.ImageURLs {
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

//
// =========================
// 🔴 SUPPRIMER UNE IMAGE
// =========================
//
func DeleteProductImage(c *gin.Context) {
	ctx := context.Background()
	imageID := c.Param("id") // ex: products/172928721234-tournevis.jpg

	// Supprime l'objet de MinIO
	err := database.MinioClient.RemoveObject(
		ctx,
		os.Getenv("MINIO_BUCKET"),
		imageID,
		minio.RemoveObjectOptions{},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur suppression MinIO"})
		return
	}

	// Supprime aussi l’URL du champ image_urls dans MongoDB
	imageURL := fmt.Sprintf("http://%s/%s/%s",
		os.Getenv("MINIO_ENDPOINT"),
		os.Getenv("MINIO_BUCKET"),
		imageID,
	)

	collection := database.MongoProductsDB.Collection("products")
	_, err = collection.UpdateMany(
		ctx,
		bson.M{"image_urls": imageURL},
		bson.M{"$pull": bson.M{"image_urls": imageURL}},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur suppression MongoDB"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "🗑️ Image supprimée avec succès",
		"image_id": imageID,
	})
}
