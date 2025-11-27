package user

import (
	"cedra_back_end/internal/cache"
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

// ✅ Récupère toutes les commandes de l'utilisateur connecté
func GetMyOrders(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur non authentifié"})
		return
	}

	session, err := database.GetOrdersSession()
	if err != nil {
		log.Printf("❌ Erreur session ScyllaDB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	// Récupérer les commandes depuis orders_by_user (triées par order_id DESC)
	var orders []models.Order
	iter := session.Query("SELECT order_id, payment_intent_id, items, total_price, status, created_at, updated_at FROM orders_by_user WHERE user_id = ?", userID).Iter()
	var (
		orderID         gocql.UUID
		paymentIntentID string
		itemsJSON       string
		totalPrice      float64
		status          string
		createdAt       time.Time
		updatedAt       *time.Time
	)
	for iter.Scan(&orderID, &paymentIntentID, &itemsJSON, &totalPrice, &status, &createdAt, &updatedAt) {
		var items []models.OrderItem
		if itemsJSON != "" {
			json.Unmarshal([]byte(itemsJSON), &items)
		}
		orders = append(orders, models.Order{
			ID:              orderID,
			UserID:          userID,
			PaymentIntentID: paymentIntentID,
			Items:           items,
			TotalPrice:      totalPrice,
			Status:          status,
			CreatedAt:       createdAt,
			UpdatedAt:       updatedAt,
		})
	}
	if err := iter.Close(); err != nil {
		log.Printf("⚠️ Erreur fermeture iter: %v", err)
	}

	// ✅ Enrichir avec les infos produits (optimisé avec cache Redis)
	if len(orders) > 0 {
		// Collecter tous les product_ids uniques
		productIDsMap := make(map[string]bool)
		for i := range orders {
			for j := range orders[i].Items {
				productIDsMap[orders[i].Items[j].ProductID] = true
			}
		}

		// Convertir en slice
		productIDs := make([]string, 0, len(productIDsMap))
		for productID := range productIDsMap {
			productIDs = append(productIDs, productID)
		}

		// Récupérer tous les noms depuis le cache (Redis + ScyllaDB si nécessaire)
		productNames := cache.GetProductNamesFromCache(productIDs)

		// Appliquer les noms aux items
		for i := range orders {
			for j := range orders[i].Items {
				if name, ok := productNames[orders[i].Items[j].ProductID]; ok {
					orders[i].Items[j].ProductName = name
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
	})
}

// ✅ Récupère une commande spécifique par ID
func GetOrderByID(c *gin.Context) {
	userID := c.GetString("user_id")
	orderID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur non authentifié"})
		return
	}

	session, err := database.GetOrdersSession()
	if err != nil {
		log.Printf("❌ Erreur session ScyllaDB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	orderUUID, err := uuid.Parse(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID commande invalide"})
		return
	}

	// Vérifier que la commande appartient à l'utilisateur
	var userIDDB, paymentIntentID, itemsJSON string
	var totalPrice float64
	var status string
	var createdAt time.Time
	var updatedAt *time.Time

	err = session.Query("SELECT user_id, payment_intent_id, items, total_price, status, created_at, updated_at FROM orders_by_user WHERE user_id = ? AND order_id = ?", userID, gocql.UUID(orderUUID)).Scan(
		&userIDDB, &paymentIntentID, &itemsJSON, &totalPrice, &status, &createdAt, &updatedAt)
	if err != nil || userIDDB != userID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Commande introuvable"})
		return
	}

	var items []models.OrderItem
	if itemsJSON != "" {
		json.Unmarshal([]byte(itemsJSON), &items)
	}

	order := models.Order{
		ID:              gocql.UUID(orderUUID),
		UserID:          userID,
		PaymentIntentID: paymentIntentID,
		Items:           items,
		TotalPrice:      totalPrice,
		Status:          status,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}

	c.JSON(http.StatusOK, order)
}
