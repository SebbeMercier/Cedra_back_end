package user

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/google/uuid"

	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
	"cedra_back_end/internal/services"
)

func GetCart(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
		return
	}

	ctx := context.Background()
	key := "cart:" + userID

	// 1️⃣ Récupérer le panier depuis Redis
	data, err := database.RedisClient.Get(ctx, key).Result()
	if err != nil || data == "" {
		c.JSON(http.StatusOK, gin.H{"items": []models.CartItem{}})
		return
	}

	var cart []models.CartItem
	if err := json.Unmarshal([]byte(data), &cart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur décodage panier"})
		return
	}

	for i := range cart {
		if cart[i].ImageURL != "" {
			key := extractMinIOKey(cart[i].ImageURL)
			signedURL, err := services.GenerateSignedURL(ctx, key, 24*time.Hour)
			if err == nil {
				cart[i].ImageURL = signedURL
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"items": cart})
}

func AddToCart(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
		return
	}

	var input struct {
		ProductID string `json:"productId" binding:"required"`
		Quantity  int    `json:"quantity" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides"})
		return
	}

	// 1️⃣ Valider le product_id
	productID, err := uuid.Parse(input.ProductID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID produit invalide"})
		return
	}

	session, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	var (
		name      string
		price     float64
		stock     int
		imageURLs []string
	)

	err = session.Query(
		`SELECT name, price, stock, image_urls FROM products WHERE product_id = ?`,
		gocql.UUID(productID),
	).Scan(&name, &price, &stock, &imageURLs)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produit introuvable"})
		return
	}

	if stock < input.Quantity {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Stock insuffisant"})
		return
	}

	imageURL := ""
	if len(imageURLs) > 0 {
		imageURL = imageURLs[0]
	}

	item := models.CartItem{
		ProductID: input.ProductID,
		Name:      name,
		Price:     price,
		Quantity:  input.Quantity,
		ImageURL:  imageURL,
	}

	ctx := context.Background()
	key := "cart:" + userID

	data, _ := database.RedisClient.Get(ctx, key).Result()
	var cart []models.CartItem
	if data != "" {
		_ = json.Unmarshal([]byte(data), &cart)
	}

	found := false
	for i := range cart {
		if cart[i].ProductID == item.ProductID {
			newQuantity := cart[i].Quantity + item.Quantity
			// Vérifier le stock total
			if newQuantity > stock {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Stock insuffisant pour cette quantité"})
				return
			}
			cart[i].Quantity = newQuantity
			found = true
			break
		}
	}
	if !found {
		cart = append(cart, item)
	}

	// 8️⃣ Sauvegarder dans Redis (30 jours)
	jsonData, _ := json.Marshal(cart)
	database.RedisClient.Set(ctx, key, jsonData, 30*24*time.Hour)

	c.JSON(http.StatusOK, gin.H{
		"message": "Produit ajouté au panier",
		"items":   cart,
	})
}

func RemoveFromCart(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
		return
	}

	productID := c.Param("productId")
	ctx := context.Background()
	key := "cart:" + userID

	// 1️⃣ Récupérer le panier
	data, err := database.RedisClient.Get(ctx, key).Result()
	if err != nil || data == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Panier vide"})
		return
	}

	var cart []models.CartItem
	if err := json.Unmarshal([]byte(data), &cart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lecture panier"})
		return
	}

	// 2️⃣ Filtrer le produit à supprimer
	newCart := []models.CartItem{}
	found := false
	for _, item := range cart {
		if item.ProductID != productID {
			newCart = append(newCart, item)
		} else {
			found = true
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produit non trouvé dans le panier"})
		return
	}

	// 3️⃣ Sauvegarder le nouveau panier
	if len(newCart) == 0 {
		database.RedisClient.Del(ctx, key)
	} else {
		jsonData, _ := json.Marshal(newCart)
		database.RedisClient.Set(ctx, key, jsonData, 30*24*time.Hour)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Produit supprimé du panier",
		"items":   newCart,
	})
}

func ClearCart(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
		return
	}

	ctx := context.Background()
	key := "cart:" + userID

	if err := database.RedisClient.Del(ctx, key).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors du vidage du panier"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Panier vidé avec succès"})
}

func UpdateCartQuantity(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
		return
	}

	productID := c.Param("productId")

	var input struct {
		Quantity int `json:"quantity" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quantité invalide (minimum 1)"})
		return
	}

	ctx := context.Background()
	key := "cart:" + userID

	// 1️⃣ Récupérer le panier depuis Redis
	data, err := database.RedisClient.Get(ctx, key).Result()
	if err != nil || data == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Panier vide"})
		return
	}

	var cart []models.CartItem
	if err := json.Unmarshal([]byte(data), &cart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lecture panier"})
		return
	}

	// 2️⃣ Trouver et mettre à jour le produit
	found := false
	for i := range cart {
		if cart[i].ProductID == productID {
			cart[i].Quantity = input.Quantity
			found = true
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produit introuvable dans le panier"})
		return
	}

	// 3️⃣ Sauvegarder dans Redis
	jsonData, _ := json.Marshal(cart)
	database.RedisClient.Set(ctx, key, jsonData, 30*24*time.Hour)

	c.JSON(http.StatusOK, gin.H{
		"message": "Quantité mise à jour",
		"items":   cart,
	})
}

func extractMinIOKey(url string) string {
	if idx := strings.Index(url, "/cedra-images/"); idx != -1 {
		return url[idx+len("/cedra-images/"):]
	}
	return strings.TrimPrefix(url, "/uploads/")
}
