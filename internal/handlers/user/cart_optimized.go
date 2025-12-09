package user

import (
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

const CartTTL = 30 * 24 * time.Hour // 30 jours

// GetCart récupère le panier (ultra-rapide, seulement Redis)
func GetCartOptimized(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
		return
	}

	ctx := context.Background()
	key := "cart:" + userID

	// ✅ Récupération ultra-rapide depuis Redis
	data, err := database.Redis.Get(ctx, key).Result()
	if err != nil || data == "" {
		c.JSON(http.StatusOK, gin.H{"items": []models.CartItem{}, "total": 0})
		return
	}

	var cart []models.CartItem
	if err := json.Unmarshal([]byte(data), &cart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur décodage panier"})
		return
	}

	// Calculer le total
	total := 0.0
	for _, item := range cart {
		total += item.Price * float64(item.Quantity)
	}

	c.JSON(http.StatusOK, gin.H{
		"items": cart,
		"total": total,
		"count": len(cart),
	})
}

// AddToCartOptimized ajoute un produit (optimisé avec cache)
func AddToCartOptimized(c *gin.Context) {
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

	productID, err := uuid.Parse(input.ProductID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID produit invalide"})
		return
	}

	// ✅ Récupérer les infos produit depuis ScyllaDB (optimisé)
	session, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	var (
		productIDDB gocql.UUID
		name        string
		price       float64
		stock       int
		imageURLs   []string
	)

	// Requête optimisée (seulement les champs nécessaires)
	err = session.Query(`SELECT product_id, name, price, stock, image_urls FROM products WHERE product_id = ?`, gocql.UUID(productID)).
		Scan(&productIDDB, &name, &price, &stock, &imageURLs)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produit introuvable"})
		return
	}

	// Vérifier le stock
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

	// ✅ Pipeline Redis pour optimiser les opérations
	pipe := database.Redis.Pipeline()

	// Récupérer le panier actuel
	getCmd := pipe.Get(ctx, key)
	_, err = pipe.Exec(ctx)

	var cart []models.CartItem
	if data, err := getCmd.Result(); err == nil && data != "" {
		json.Unmarshal([]byte(data), &cart)
	}

	// Mettre à jour ou ajouter l'item
	found := false
	for i := range cart {
		if cart[i].ProductID == item.ProductID {
			newQuantity := cart[i].Quantity + item.Quantity
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

	// Sauvegarder avec pipeline
	jsonData, _ := json.Marshal(cart)
	pipe = database.Redis.Pipeline()
	pipe.Set(ctx, key, jsonData, CartTTL)
	pipe.Publish(ctx, "cart:"+userID, "updated") // ✅ Pub/Sub pour sync temps réel
	pipe.Exec(ctx)

	// Calculer le total
	total := 0.0
	for _, item := range cart {
		total += item.Price * float64(item.Quantity)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Produit ajouté au panier",
		"items":   cart,
		"total":   total,
		"count":   len(cart),
	})
}

// UpdateCartQuantityOptimized met à jour la quantité (ultra-rapide)
func UpdateCartQuantityOptimized(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
		return
	}

	productID := c.Param("productId")

	var input struct {
		Quantity int `json:"quantity" binding:"required,min=0"` // 0 = supprimer
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quantité invalide"})
		return
	}

	ctx := context.Background()
	key := "cart:" + userID

	// Récupérer le panier
	data, err := database.Redis.Get(ctx, key).Result()
	if err != nil || data == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Panier vide"})
		return
	}

	var cart []models.CartItem
	if err := json.Unmarshal([]byte(data), &cart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lecture panier"})
		return
	}

	// Mettre à jour ou supprimer
	newCart := []models.CartItem{}
	found := false
	for i := range cart {
		if cart[i].ProductID == productID {
			found = true
			if input.Quantity > 0 {
				cart[i].Quantity = input.Quantity
				newCart = append(newCart, cart[i])
			}
			// Si quantity = 0, on ne l'ajoute pas (suppression)
		} else {
			newCart = append(newCart, cart[i])
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produit introuvable dans le panier"})
		return
	}

	// Sauvegarder avec pub/sub
	pipe := database.Redis.Pipeline()
	if len(newCart) == 0 {
		pipe.Del(ctx, key)
	} else {
		jsonData, _ := json.Marshal(newCart)
		pipe.Set(ctx, key, jsonData, CartTTL)
	}
	pipe.Publish(ctx, "cart:"+userID, "updated")
	pipe.Exec(ctx)

	// Calculer le total
	total := 0.0
	for _, item := range newCart {
		total += item.Price * float64(item.Quantity)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Quantité mise à jour",
		"items":   newCart,
		"total":   total,
		"count":   len(newCart),
	})
}

// RemoveFromCartOptimized supprime un produit (ultra-rapide)
func RemoveFromCartOptimized(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
		return
	}

	productID := c.Param("productId")
	ctx := context.Background()
	key := "cart:" + userID

	// Récupérer le panier
	data, err := database.Redis.Get(ctx, key).Result()
	if err != nil || data == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Panier vide"})
		return
	}

	var cart []models.CartItem
	if err := json.Unmarshal([]byte(data), &cart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lecture panier"})
		return
	}

	// Filtrer le produit
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

	// Sauvegarder avec pub/sub
	pipe := database.Redis.Pipeline()
	if len(newCart) == 0 {
		pipe.Del(ctx, key)
	} else {
		jsonData, _ := json.Marshal(newCart)
		pipe.Set(ctx, key, jsonData, CartTTL)
	}
	pipe.Publish(ctx, "cart:"+userID, "updated")
	pipe.Exec(ctx)

	// Calculer le total
	total := 0.0
	for _, item := range newCart {
		total += item.Price * float64(item.Quantity)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Produit supprimé du panier",
		"items":   newCart,
		"total":   total,
		"count":   len(newCart),
	})
}

// ClearCartOptimized vide le panier (ultra-rapide)
func ClearCartOptimized(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
		return
	}

	ctx := context.Background()
	key := "cart:" + userID

	// Supprimer avec pub/sub
	pipe := database.Redis.Pipeline()
	pipe.Del(ctx, key)
	pipe.Publish(ctx, "cart:"+userID, "cleared")
	pipe.Exec(ctx)

	c.JSON(http.StatusOK, gin.H{
		"message": "Panier vidé avec succès",
		"items":   []models.CartItem{},
		"total":   0,
		"count":   0,
	})
}

// SyncCart endpoint pour synchroniser le panier entre app et web
func SyncCart(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
		return
	}

	ctx := context.Background()
	key := "cart:" + userID

	// Récupérer le panier actuel
	data, err := database.Redis.Get(ctx, key).Result()
	if err != nil || data == "" {
		c.JSON(http.StatusOK, gin.H{
			"items":  []models.CartItem{},
			"total":  0,
			"count":  0,
			"synced": true,
		})
		return
	}

	var cart []models.CartItem
	if err := json.Unmarshal([]byte(data), &cart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur décodage panier"})
		return
	}

	// Calculer le total
	total := 0.0
	for _, item := range cart {
		total += item.Price * float64(item.Quantity)
	}

	c.JSON(http.StatusOK, gin.H{
		"items":  cart,
		"total":  total,
		"count":  len(cart),
		"synced": true,
	})
}
