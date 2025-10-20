package handlers

import (
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetCart(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "non authentifié"})
		return
	}

	key := "cart:" + userID
	data, err := database.RedisClient.Get(context.Background(), key).Result()
	if err != nil || data == "" {
		c.JSON(http.StatusOK, gin.H{"items": []models.CartItem{}}) // panier vide
		return
	}

	var cart []models.CartItem
	if err := json.Unmarshal([]byte(data), &cart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur décodage panier"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": cart})
}

//
// 🟢 POST /api/cart/add
//
func AddToCart(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "non authentifié"})
		return
	}

	key := "cart:" + userID

	var input struct {
		ProductID string `json:"productId"`
		Quantity  int    `json:"quantity"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides"})
		return
	}

	if input.Quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quantité invalide"})
		return
	}

	// 🧩 Récupération du produit depuis MongoDB
	coll := database.MongoProductsDB.Collection("products")
	var product models.Product

	objID, err := primitive.ObjectIDFromHex(input.ProductID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID produit invalide"})
		return
	}

	if err := coll.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&product); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produit introuvable"})
		return
	}

	// 🖼️ Première image pour l’aperçu panier
	imageURL := ""
	if len(product.ImageURLs) > 0 {
		imageURL = product.ImageURLs[0]
	}

	// 🔹 Création de l’item
	item := models.CartItem{
		ProductID: input.ProductID,
		Name:      product.Name,
		Price:     product.Price,
		Quantity:  input.Quantity,
		ImageURL:  imageURL,
	}

	// 🧠 Récupère panier actuel depuis Redis
	data, _ := database.RedisClient.Get(context.Background(), key).Result()
	var cart []models.CartItem
	if data != "" {
		_ = json.Unmarshal([]byte(data), &cart)
	}

	// 🔁 Met à jour ou ajoute l’item
	found := false
	for i := range cart {
		if cart[i].ProductID == item.ProductID {
			cart[i].Quantity += item.Quantity
			found = true
			break
		}
	}
	if !found {
		cart = append(cart, item)
	}

	// 💾 Sauvegarde dans Redis (30 jours)
	jsonData, _ := json.Marshal(cart)
	database.RedisClient.Set(context.Background(), key, jsonData, 30*24*time.Hour)

	c.JSON(http.StatusOK, gin.H{
		"message": "Produit ajouté au panier",
		"items":   cart,
	})
}

//
// ❌ DELETE /api/cart/:productId
//
func RemoveFromCart(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "non authentifié"})
		return
	}

	productID := c.Param("productId")
	key := "cart:" + userID

	data, _ := database.RedisClient.Get(context.Background(), key).Result()
	if data == "" {
		c.JSON(http.StatusOK, gin.H{"message": "Panier vide"})
		return
	}

	var cart []models.CartItem
	_ = json.Unmarshal([]byte(data), &cart)

	newCart := []models.CartItem{}
	for _, item := range cart {
		if item.ProductID != productID {
			newCart = append(newCart, item)
		}
	}

	jsonData, _ := json.Marshal(newCart)
	database.RedisClient.Set(context.Background(), key, jsonData, 30*24*time.Hour)

	c.JSON(http.StatusOK, gin.H{
		"message": "Produit supprimé du panier",
		"items":   newCart,
	})
}

//
// 🧹 DELETE /api/cart/clear
//
func ClearCart(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "non authentifié"})
		return
	}

	key := "cart:" + userID

	// 🧹 Supprime complètement la clé Redis
	if err := database.RedisClient.Del(context.Background(), key).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors du vidage du panier"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Panier vidé avec succès",
	})
}