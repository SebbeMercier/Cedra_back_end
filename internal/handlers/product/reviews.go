package product

import (
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

// CreateReview crée un avis sur un produit
func CreateReview(c *gin.Context) {
	userID := c.GetString("user_id")
	productID := c.Param("id")

	var req struct {
		Rating  int    `json:"rating" binding:"required,min=1,max=5"`
		Comment string `json:"comment" binding:"required,min=10,max=500"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides", "details": err.Error()})
		return
	}

	productUUID, err := uuid.Parse(productID)
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

	// Vérifier que l'utilisateur a acheté ce produit
	ordersSession, err := database.GetOrdersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	// Vérifier dans orders_by_user
	iter := ordersSession.Query("SELECT items FROM orders_by_user WHERE user_id = ?", userID).Iter()
	var hasPurchased bool
	var itemsJSON string

	for iter.Scan(&itemsJSON) {
		// Vérifier si le produit est dans les items
		if len(itemsJSON) > 0 && contains(itemsJSON, productID) {
			hasPurchased = true
			break
		}
	}
	iter.Close()

	if !hasPurchased {
		c.JSON(http.StatusForbidden, gin.H{"error": "Vous devez avoir acheté ce produit pour laisser un avis"})
		return
	}

	// Récupérer le nom de l'utilisateur
	usersSession, err := database.GetUsersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	var userName string
	err = usersSession.Query("SELECT name FROM users WHERE user_id = ?", userID).Scan(&userName)
	if err != nil || userName == "" {
		userName = "Utilisateur"
	}

	// Créer l'avis
	reviewID := gocql.TimeUUID()
	now := time.Now()

	err = productsSession.Query(`
		INSERT INTO reviews (review_id, product_id, user_id, user_name, rating, comment, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, reviewID, gocql.UUID(productUUID), userID, userName, req.Rating, req.Comment, now).Exec()

	if err != nil {
		log.Printf("❌ Erreur création avis: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur création avis"})
		return
	}

	// Créer l'index reviews_by_product
	err = productsSession.Query(`
		INSERT INTO reviews_by_product (product_id, review_id, user_id, user_name, rating, comment, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, gocql.UUID(productUUID), reviewID, userID, userName, req.Rating, req.Comment, now).Exec()

	if err != nil {
		log.Printf("⚠️ Erreur index reviews_by_product: %v", err)
	}

	log.Printf("⭐ Avis créé: %s pour produit %s (note: %d/5)", reviewID, productID, req.Rating)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Avis créé avec succès",
		"review": models.Review{
			ID:        reviewID,
			ProductID: gocql.UUID(productUUID),
			UserID:    userID,
			UserName:  userName,
			Rating:    req.Rating,
			Comment:   req.Comment,
			CreatedAt: now,
		},
	})
}

// GetProductReviews récupère les avis d'un produit
func GetProductReviews(c *gin.Context) {
	productID := c.Param("id")

	productUUID, err := uuid.Parse(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID produit invalide"})
		return
	}

	session, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	// Récupérer les avis
	iter := session.Query(`
		SELECT review_id, user_id, user_name, rating, comment, created_at
		FROM reviews_by_product WHERE product_id = ?
	`, gocql.UUID(productUUID)).Iter()

	var reviews []models.Review
	var review models.Review
	var totalRating int

	for iter.Scan(&review.ID, &review.UserID, &review.UserName, &review.Rating, &review.Comment, &review.CreatedAt) {
		review.ProductID = gocql.UUID(productUUID)
		reviews = append(reviews, review)
		totalRating += review.Rating
	}

	if err := iter.Close(); err != nil {
		log.Printf("❌ Erreur lecture avis: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lecture avis"})
		return
	}

	// Calculer la moyenne
	var averageRating float64
	if len(reviews) > 0 {
		averageRating = float64(totalRating) / float64(len(reviews))
	}

	c.JSON(http.StatusOK, gin.H{
		"reviews":        reviews,
		"total_reviews":  len(reviews),
		"average_rating": averageRating,
	})
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
