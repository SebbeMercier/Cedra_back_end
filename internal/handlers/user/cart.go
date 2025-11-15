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

	// 🧩 Récupération du produit depuis ScyllaDB
	session, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	productID, err := uuid.Parse(input.ProductID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID produit invalide"})
		return
	}
	productUUID := gocql.UUID(productID)

	var product models.Product
	var (
		name, description string
		price             float64
		stock             int
		categoryID        gocql.UUID
		imageURLs         []string
		tags              []string
		companyID         gocql.UUID
		createdAt, updatedAt *time.Time
	)

	err = session.Query(`SELECT product_id, name, description, price, stock, category_id, company_id, image_urls, tags, created_at, updated_at 
	                     FROM products WHERE product_id = ?`, productUUID).Scan(
		&product.ID, &name, &description, &price, &stock, &categoryID, &companyID, &imageURLs, &tags, &createdAt, &updatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produit introuvable"})
		return
	}

	product.Name = name
	product.Description = description
	product.Price = price
	product.Stock = stock
	product.CategoryID = categoryID
	product.ImageURLs = imageURLs
	product.Tags = tags
	product.CompanyID = companyID
	product.CreatedAt = createdAt
	product.UpdatedAt = updatedAt

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