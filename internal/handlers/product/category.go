package product

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func getCategoryCollection() *mongo.Collection {
	if database.MongoCategoriesDB == nil {
		panic("❌ MongoCategoriesDB n'est pas initialisée")
	}
	return database.MongoCategoriesDB.Collection("categories")
}

// 🟢 Créer une catégorie
func CreateCategory(c *gin.Context) {
	var cat models.Category
	if err := c.ShouldBindJSON(&cat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if cat.Name == "" || cat.Slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Les champs 'name' et 'slug' sont obligatoires"})
		return
	}

	res, err := getCategoryCollection().InsertOne(context.TODO(), cat)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": res.InsertedID})
}

// 🔵 Lister les catégories
func GetAllCategories(c *gin.Context) {
	ctx := context.Background()
	cacheKey := "categories:all"

	// Cache Redis
	val, err := database.RedisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		var cached []models.Category
		json.Unmarshal([]byte(val), &cached)
		c.JSON(http.StatusOK, cached)
		return
	}

	cursor, err := getCategoryCollection().Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var cats []models.Category
	cursor.All(ctx, &cats)

	data, _ := json.Marshal(cats)
	database.RedisClient.Set(ctx, cacheKey, data, time.Hour)

	c.JSON(http.StatusOK, cats)
}
