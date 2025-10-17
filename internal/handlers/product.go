package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
	"cedra_back_end/internal/service"
)

//
// --- MONGO COLLECTION ---
//
func getProductCollection() *mongo.Collection {
	if database.MongoProductsDB == nil {
		panic("❌ MongoProductsDB non initialisée — as-tu bien appelé database.ConnectDatabases() ?")
	}
	return database.MongoProductsDB.Collection("products")
}

//
// --- HANDLERS ---
//

// 🟢 Créer un produit (admin)
// 🟢 Créer un produit (avec vérification de la catégorie)
func CreateProduct(c *gin.Context) {
	var p models.Product
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if p.CategoryID.IsZero() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le champ 'category_id' est obligatoire"})
		return
	}

	// ✅ Vérifie que la catégorie existe dans db_categories
	ctx := context.Background()
	catColl := database.MongoCategoriesDB.Collection("categories")

	var category bson.M
	err := catColl.FindOne(ctx, bson.M{"_id": p.CategoryID}).Decode(&category)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Catégorie introuvable"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur vérification catégorie"})
		return
	}

	res, err := getProductCollection().InsertOne(ctx, p)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	p.ID = res.InsertedID.(primitive.ObjectID)

	// 🔄 Indexe dans Elasticsearch
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

// 🔍 Recherche de produits via Elasticsearch ou Mongo si indisponible
func SearchProducts(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "paramètre 'q' manquant"})
		return
	}

	// 1️⃣ Tentative de recherche dans Elasticsearch
	results, err := services.SearchProducts(query)
	if err == nil && len(results) > 0 {
		c.JSON(http.StatusOK, results)
		return
	}

	// 2️⃣ Si Elasticsearch est vide ou indisponible → fallback MongoDB
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

	if len(products) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "Aucun produit trouvé"})
		return
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