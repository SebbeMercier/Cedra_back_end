package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"cedra_back_end/internal/database"
)

type ProductImage struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ProductID  primitive.ObjectID `bson:"product_id" json:"product_id"`
	URL        string             `bson:"url" json:"url"`
	FileName   string             `bson:"file_name" json:"file_name"`
	UploadedAt time.Time          `bson:"uploaded_at" json:"uploaded_at"`
	UserID     string             `bson:"user_id" json:"user_id"`
}

// === POST /api/images/upload ===

func UploadProductImage(c *gin.Context) {
	ctx := context.Background()
	bucket := os.Getenv("MINIO_BUCKET")

	// 🔒 Authentification
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
		return
	}

	// ✅ Récupère le product_id dans le formulaire
	productIDStr := c.PostForm("product_id")
	if productIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le champ 'product_id' est requis"})
		return
	}
	productID, err := primitive.ObjectIDFromHex(productIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de produit invalide"})
		return
	}

	// ✅ Récupère le fichier uploadé
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Aucun fichier reçu"})
		return
	}

	// ✅ Ouvre le fichier (pas de stockage temporaire)
	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur ouverture fichier"})
		return
	}
	defer file.Close()

	// ✅ Nom unique du fichier
	objectName := fmt.Sprintf("products/%s_%d%s", productID.Hex(), time.Now().Unix(), filepath.Ext(fileHeader.Filename))

	// ✅ Upload vers MinIO
	info, err := database.MinioClient.PutObject(
		ctx,
		bucket,
		objectName,
		file,
		fileHeader.Size,
		minio.PutObjectOptions{ContentType: fileHeader.Header.Get("Content-Type")},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur upload MinIO", "details": err.Error()})
		return
	}

	// ✅ URL publique (à adapter selon ton reverse proxy)
	publicBase := os.Getenv("MINIO_PUBLIC_URL")
	if publicBase == "" {
		publicBase = fmt.Sprintf("http://%s", os.Getenv("MINIO_ENDPOINT"))
	}
	publicURL := fmt.Sprintf("%s/%s/%s", publicBase, bucket, objectName)

	// ✅ Enregistre dans MongoDB
	col := database.MongoProductsDB.Collection("images")
	imgDoc := ProductImage{
		ProductID:  productID,
		URL:        publicURL,
		FileName:   info.Key,
		UploadedAt: time.Now(),
		UserID:     userID,
	}

	res, err := col.InsertOne(ctx, imgDoc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur insertion MongoDB", "details": err.Error()})
		return
	}

	imgDoc.ID = res.InsertedID.(primitive.ObjectID)

	c.JSON(http.StatusOK, gin.H{
		"message": "✅ Image uploadée et liée au produit",
		"image":   imgDoc,
	})
}

// === GET /api/images/:productId ===
func GetProductImages(c *gin.Context) {
	productIDStr := c.Param("productId")

	// 🔸 Vérifie l'ID produit
	productID, err := primitive.ObjectIDFromHex(productIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID produit invalide"})
		return
	}

	col := database.MongoProductsDB.Collection("images")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 🔹 Vérifie que le produit existe (optionnel mais plus propre)
	prodColl := database.MongoProductsDB.Collection("products")
	var exists bson.M
	if err := prodColl.FindOne(ctx, bson.M{"_id": productID}).Decode(&exists); err != nil {
		if err.Error() == "mongo: no documents in result" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Produit introuvable"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur vérification produit"})
		return
	}

	// 🔹 Recherche des images liées
	cursor, err := col.Find(ctx, bson.M{"product_id": productID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lecture MongoDB"})
		return
	}
	defer cursor.Close(ctx)

	var results []ProductImage
	if err := cursor.All(ctx, &results); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur décodage images"})
		return
	}

	// 🔹 Si aucune image → message clair
	if len(results) == 0 {
		c.JSON(http.StatusOK, gin.H{"images": []ProductImage{}, "message": "Aucune image trouvée pour ce produit"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"images": results})
}

func DeleteProductImage(c *gin.Context) {
	ctx := context.Background()
	imageIDStr := c.Param("id")
	bucket := os.Getenv("MINIO_BUCKET")

	// 🔒 Vérifie l'authentification
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
		return
	}

	// 🔒 Vérifie si l'utilisateur est admin (si tu as un middleware spécial)
	isAdmin := c.GetBool("is_admin")
	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Accès réservé aux administrateurs"})
		return
	}

	// ✅ Vérifie l'ID
	imgID, err := primitive.ObjectIDFromHex(imageIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID d'image invalide"})
		return
	}

	col := database.MongoProductsDB.Collection("images")

	// ✅ Récupère le document avant suppression
	var img ProductImage
	err = col.FindOne(ctx, bson.M{"_id": imgID}).Decode(&img)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image introuvable"})
		return
	}

	// ✅ Supprime du bucket MinIO
	err = database.MinioClient.RemoveObject(ctx, bucket, img.FileName, minio.RemoveObjectOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur suppression MinIO", "details": err.Error()})
		return
	}

	// ✅ Supprime le document MongoDB
	_, err = col.DeleteOne(ctx, bson.M{"_id": imgID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur suppression MongoDB", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "🗑️ Image supprimée avec succès",
		"image_url": img.URL,
	})
}
