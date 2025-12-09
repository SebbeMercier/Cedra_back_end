package user

import (
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

// GetWishlist récupère la wishlist de l'utilisateur
func GetWishlist(c *gin.Context) {
	userID := c.GetString("user_id")

	// Récupérer depuis Redis d'abord
	ctx := context.Background()
	cacheKey := "wishlist:" + userID

	cached, err := database.Redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var wishlist models.Wishlist
		if json.Unmarshal([]byte(cached), &wishlist) == nil {
			c.JSON(http.StatusOK, wishlist)
			return
		}
	}

	// Sinon depuis ScyllaDB
	session, err := database.GetUsersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	iter := session.Query("SELECT product_id FROM wishlist WHERE user_id = ?", userID).Iter()

	var productIDs []gocql.UUID
	var productID gocql.UUID

	for iter.Scan(&productID) {
		productIDs = append(productIDs, productID)
	}

	if err := iter.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lecture wishlist"})
		return
	}

	// Récupérer les détails des produits
	productsSession, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	var products []models.Product
	for _, pid := range productIDs {
		var product models.Product
		err := productsSession.Query(`
			SELECT product_id, name, description, price, stock, category_id, image_urls, tags, created_at, updated_at
			FROM products WHERE product_id = ?
		`, pid).Scan(
			&product.ID, &product.Name, &product.Description, &product.Price,
			&product.Stock, &product.CategoryID, &product.ImageURLs, &product.Tags,
			&product.CreatedAt, &product.UpdatedAt,
		)
		if err == nil {
			products = append(products, product)
		}
	}

	wishlist := models.Wishlist{
		UserID: userID,
		Items:  products,
	}

	// Mettre en cache
	if data, err := json.Marshal(wishlist); err == nil {
		database.Redis.Set(ctx, cacheKey, data, 10*time.Minute)
	}

	c.JSON(http.StatusOK, wishlist)
}

// AddToWishlist ajoute un produit à la wishlist
func AddToWishlist(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		ProductID string `json:"product_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides"})
		return
	}

	productUUID, err := uuid.Parse(req.ProductID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID produit invalide"})
		return
	}

	// Vérifier que le produit existe
	productsSession, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	var exists bool
	err = productsSession.Query("SELECT product_id FROM products WHERE product_id = ?", gocql.UUID(productUUID)).Scan(&productUUID)
	exists = (err == nil)

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produit introuvable"})
		return
	}

	// Ajouter à la wishlist
	session, err := database.GetUsersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	err = session.Query(`
		INSERT INTO wishlist (user_id, product_id, added_at)
		VALUES (?, ?, ?)
	`, userID, gocql.UUID(productUUID), time.Now()).Exec()

	if err != nil {
		log.Printf("❌ Erreur ajout wishlist: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur ajout à la wishlist"})
		return
	}

	// Invalider le cache
	ctx := context.Background()
	database.Redis.Del(ctx, "wishlist:"+userID)

	log.Printf("⭐ Produit %s ajouté à la wishlist de %s", req.ProductID, userID)

	c.JSON(http.StatusOK, gin.H{
		"message":    "Produit ajouté à la wishlist",
		"product_id": req.ProductID,
	})
}

// RemoveFromWishlist retire un produit de la wishlist
func RemoveFromWishlist(c *gin.Context) {
	userID := c.GetString("user_id")
	productID := c.Param("productId")

	productUUID, err := uuid.Parse(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID produit invalide"})
		return
	}

	session, err := database.GetUsersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	err = session.Query("DELETE FROM wishlist WHERE user_id = ? AND product_id = ?",
		userID, gocql.UUID(productUUID)).Exec()

	if err != nil {
		log.Printf("❌ Erreur suppression wishlist: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur suppression de la wishlist"})
		return
	}

	// Invalider le cache
	ctx := context.Background()
	database.Redis.Del(ctx, "wishlist:"+userID)

	log.Printf("🗑️ Produit %s retiré de la wishlist de %s", productID, userID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Produit retiré de la wishlist",
	})
}
