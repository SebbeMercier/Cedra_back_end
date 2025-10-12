package handlers

import (
	"cedra_back_end/internal/database"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type CartItem struct {
	ProductID string  `json:"productId"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Quantity  int     `json:"quantity"`
	ImageURL  string  `json:"image_url"`
}

// 🟢 GET /api/cart
func GetCart(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "non authentifié"})
		return
	}

	key := "cart:" + userID
	data, err := database.RedisClient.Get(context.Background(), key).Result()
	if err != nil {
		c.JSON(http.StatusOK, []CartItem{}) // Panier vide par défaut
		return
	}

	var cart []CartItem
	if err := json.Unmarshal([]byte(data), &cart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur décodage panier"})
		return
	}

	c.JSON(http.StatusOK, cart)
}

// 🟢 POST /api/cart/add
func AddToCart(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "non authentifié"})
		return
	}

	key := "cart:" + userID

	var item CartItem
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides"})
		return
	}

	data, _ := database.RedisClient.Get(context.Background(), key).Result()
	var cart []CartItem
	if data != "" {
		_ = json.Unmarshal([]byte(data), &cart)
	}

	// Vérifie si le produit est déjà présent
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

	jsonData, _ := json.Marshal(cart)
	database.RedisClient.Set(context.Background(), key, jsonData, 30*24*time.Hour)

	c.JSON(http.StatusOK, cart)
}

// 🟢 DELETE /api/cart/:productId
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

	var cart []CartItem
	_ = json.Unmarshal([]byte(data), &cart)

	newCart := []CartItem{}
	for _, item := range cart {
		if item.ProductID != productID {
			newCart = append(newCart, item)
		}
	}

	jsonData, _ := json.Marshal(newCart)
	database.RedisClient.Set(context.Background(), key, jsonData, 30*24*time.Hour)

	c.JSON(http.StatusOK, newCart)
}
