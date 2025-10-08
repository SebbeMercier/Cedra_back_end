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

	// ✅ Ajoute le produit
	p.CreatedAt = time.Now()
	p.UpdatedAt = p.CreatedAt

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


// 🔵 Lister les produits
func GetAllProducts(c *gin.Context) {
	ctx := context.Background()
	cacheKey := "products:all"

	// ✅ Vérifie le cache Redis
	if val, err := database.RedisClient.Get(ctx, cacheKey).Result(); err == nil && val != "" {
		var cached []models.Product
		if err := json.Unmarshal([]byte(val), &cached); err == nil {
			c.JSON(http.StatusOK, cached)
			return
		}
	}

	// 🧠 Sinon, lit depuis Mongo
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

	// 🔁 Met en cache Redis pour 1h
	if data, err := json.Marshal(products); err == nil {
		database.RedisClient.Set(ctx, cacheKey, data, time.Hour)
	}

	c.JSON(http.StatusOK, products)
}

// 🔍 Recherche de produits via Elasticsearch
func SearchProducts(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "paramètre 'q' manquant"})
		return
	}

	results, err := services.SearchProducts(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, results)
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


// 🛒 Ajouter un produit au panier (Redis)
func AddToCart(c *gin.Context) {
	userID := c.GetString("user_id") // défini par middleware JWT
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur non authentifié"})
		return
	}

	var item models.CartItem
	if err := c.BindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	cartKey := "cart:" + userID

	val, _ := database.RedisClient.Get(ctx, cartKey).Result()
	var cart models.Cart
	if val != "" {
		_ = json.Unmarshal([]byte(val), &cart)
	} else {
		cart.UserID = userID
	}

	found := false
	for i, ci := range cart.Items {
		if ci.ProductID == item.ProductID {
			cart.Items[i].Quantity += item.Quantity
			found = true
			break
		}
	}
	if !found {
		cart.Items = append(cart.Items, item)
	}

	data, _ := json.Marshal(cart)
	database.RedisClient.Set(ctx, cartKey, data, 24*time.Hour)

	c.JSON(http.StatusOK, gin.H{"message": "Produit ajouté au panier"})
}

// 🧺 Voir le panier
func GetCart(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur non authentifié"})
		return
	}

	val, err := database.RedisClient.Get(context.Background(), "cart:"+userID).Result()
	if err != nil || val == "" {
		c.JSON(http.StatusOK, gin.H{"items": []models.CartItem{}})
		return
	}

	var cart models.Cart
	_ = json.Unmarshal([]byte(val), &cart)
	c.JSON(http.StatusOK, cart)
}
