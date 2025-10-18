package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
	"cedra_back_end/internal/services"
)


func getProductCollection() *mongo.Collection {
	if database.MongoProductsDB == nil {
		panic("❌ MongoProductsDB non initialisée — as-tu bien appelé database.ConnectDatabases() ?")
	}
	return database.MongoProductsDB.Collection("products")
}

func CreateProduct(c *gin.Context) {
	ctx := context.Background()
	var p models.Product

	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if p.CategoryID.IsZero() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le champ 'category_id' est obligatoire"})
		return
	}

	// ✅ Vérifie la catégorie
	catColl := database.MongoCategoriesDB.Collection("categories")
	var category bson.M
	if err := catColl.FindOne(ctx, bson.M{"_id": p.CategoryID}).Decode(&category); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Catégorie introuvable"})
		return
	}

	// ✅ Génère automatiquement l’URL MinIO si tu sais où l’image est stockée
	// Exemple : ton upload MinIO met le fichier dans "products/xxx.jpg"
	if len(p.ImageURLs) == 0 || p.ImageURLs[0] == "" {
		imageURL := fmt.Sprintf("http://%s/%s/products/%s.jpg",
			os.Getenv("MINIO_ENDPOINT"),
			os.Getenv("MINIO_BUCKET"),
			p.Name, // tu peux adapter selon ta logique
		)
		p.ImageURLs = []string{imageURL}
	}

	// ✅ Sauvegarde dans Mongo
	res, err := getProductCollection().InsertOne(ctx, p)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	p.ID = res.InsertedID.(primitive.ObjectID)

	// 🔄 Indexation Elasticsearch
	go services.IndexProduct(p)

	c.JSON(http.StatusOK, p)
}


func GetAllProducts(c *gin.Context) {
	ctx := context.Background()
	cacheKey := "products:all"

	if val, err := database.RedisClient.Get(ctx, cacheKey).Result(); err == nil && val != "" {
		var cached []models.Product
		if err := json.Unmarshal([]byte(val), &cached); err == nil {
			c.JSON(http.StatusOK, cached)
			return
		}
	}

	cursor, err := getProductCollection().Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var products []models.Product
	if err := cursor.All(ctx, &products); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

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

	// 🔎 1️⃣ Recherche dans Elasticsearch
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

	// 🔁 2️⃣ Fallback MongoDB si ES vide
	ctx := context.Background()
	filter := bson.M{
		"$or": []bson.M{
			{"name": bson.M{"$regex": query, "$options": "i"}},
			{"description": bson.M{"$regex": query, "$options": "i"}},
			{"tags": bson.M{"$regex": query, "$options": "i"}},
		},
	}

	cursor, err := getProductCollection().Find(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur recherche MongoDB"})
		return
	}

	var products []models.Product
	if err := cursor.All(ctx, &products); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// ✅ Génère les URLs signées MinIO
	for i, p := range products {
		signed := []string{}
		for _, url := range p.ImageURLs {
			signedURL, err := services.GenerateSignedURL(ctx, url, 24*time.Hour)
			if err == nil {
				signed = append(signed, signedURL)
			}
		}
		products[i].ImageURLs = signed
	}

	c.JSON(http.StatusOK, products)
}


func GetProductsByCategory(c *gin.Context) {
	categoryID := c.Param("id")

	objID, err := primitive.ObjectIDFromHex(categoryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de catégorie invalide"})
		return
	}

	ctx := context.Background()
	cursor, err := getProductCollection().Find(ctx, bson.M{"category_id": objID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var products []models.Product
	cursor.All(ctx, &products)
	c.JSON(http.StatusOK, products)
}