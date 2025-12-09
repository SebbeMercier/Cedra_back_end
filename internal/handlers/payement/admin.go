package pa

import (
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
	"cedra_back_end/internal/utils"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

// UpdateOrderStatus permet à un admin de mettre à jour le statut d'une commande
func UpdateOrderStatus(c *gin.Context) {
	orderID := c.Param("id")
	
	var req struct {
		Status         string `json:"status" binding:"required"`
		TrackingNumber string `json:"tracking_number"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides", "details": err.Error()})
		return
	}

	// Valider le statut
	validStatuses := map[string]bool{
		"pending":   true,
		"paid":      true,
		"shipped":   true,
		"delivered": true,
		"cancelled": true,
	}

	if !validStatuses[req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Statut invalide",
			"valid_statuses": []string{"pending", "paid", "shipped", "delivered", "cancelled"},
		})
		return
	}

	orderUUID, err := uuid.Parse(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID commande invalide"})
		return
	}

	session, err := database.GetOrdersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	// Vérifier que la commande existe et récupérer le user_id
	var userID string
	err = session.Query("SELECT user_id FROM orders WHERE order_id = ?", gocql.UUID(orderUUID)).Scan(&userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Commande introuvable"})
		return
	}

	now := time.Now()

	// Mettre à jour dans orders
	query := "UPDATE orders SET status = ?, updated_at = ? WHERE order_id = ?"
	err = session.Query(query, req.Status, now, gocql.UUID(orderUUID)).Exec()
	if err != nil {
		log.Printf("❌ Erreur mise à jour orders: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur mise à jour commande"})
		return
	}

	// Mettre à jour dans orders_by_user
	query = "UPDATE orders_by_user SET status = ? WHERE user_id = ? AND order_id = ?"
	err = session.Query(query, req.Status, userID, gocql.UUID(orderUUID)).Exec()
	if err != nil {
		log.Printf("⚠️ Erreur mise à jour orders_by_user: %v", err)
	}

	log.Printf("✅ Commande %s mise à jour: %s", orderID, req.Status)

	// Envoyer une notification email à l'utilisateur
	usersSession, err := database.GetUsersSession()
	if err == nil {
		var userEmail string
		err = usersSession.Query("SELECT email FROM users WHERE user_id = ?", userID).Scan(&userEmail)
		if err == nil && userEmail != "" {
			// Créer un objet order pour l'email
			order := models.Order{
				ID:         gocql.UUID(orderUUID),
				UserID:     userID,
				Status:     req.Status,
			}
			
			// Récupérer le total de la commande
			var totalPrice float64
			session.Query("SELECT total_price FROM orders WHERE order_id = ?", gocql.UUID(orderUUID)).Scan(&totalPrice)
			order.TotalPrice = totalPrice

			// Envoyer l'email de notification (async)
			go func() {
				if err := utils.SendOrderStatusEmail(order, userEmail, req.Status); err != nil {
					log.Printf("⚠️ Erreur envoi email notification: %v", err)
				}
			}()
		}
	}

	response := gin.H{
		"success": true,
		"order_id": orderID,
		"status": req.Status,
		"updated_at": now,
	}

	if req.TrackingNumber != "" {
		response["tracking_number"] = req.TrackingNumber
	}

	c.JSON(http.StatusOK, response)
}

// GetAllOrders permet à un admin de récupérer toutes les commandes
func GetAllOrders(c *gin.Context) {
	session, err := database.GetOrdersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	// Récupérer toutes les commandes (attention: peut être lourd en production)
	iter := session.Query("SELECT order_id, user_id, payment_intent_id, items, total_price, status, created_at, updated_at FROM orders").Iter()

	type OrderResponse struct {
		ID              string    `json:"id"`
		UserID          string    `json:"user_id"`
		PaymentIntentID string    `json:"payment_intent_id"`
		Items           string    `json:"items"`
		TotalPrice      float64   `json:"total_price"`
		Status          string    `json:"status"`
		CreatedAt       time.Time `json:"created_at"`
		UpdatedAt       *time.Time `json:"updated_at,omitempty"`
	}

	var orders []OrderResponse
	var order OrderResponse
	var orderID gocql.UUID
	
	for iter.Scan(&orderID, &order.UserID, &order.PaymentIntentID, &order.Items, &order.TotalPrice, &order.Status, &order.CreatedAt, &order.UpdatedAt) {
		order.ID = orderID.String()
		orders = append(orders, order)
	}

	if err := iter.Close(); err != nil {
		log.Printf("❌ Erreur lecture commandes: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lecture commandes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"count": len(orders),
	})
}

// GetOrderStats retourne des statistiques sur les commandes
func GetOrderStats(c *gin.Context) {
	session, err := database.GetOrdersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	// Compter les commandes par statut
	stats := make(map[string]int)
	var totalRevenue float64
	var totalOrders int

	iter := session.Query("SELECT status, total_price FROM orders").Iter()
	
	var status string
	var price float64
	
	for iter.Scan(&status, &price) {
		stats[status]++
		totalRevenue += price
		totalOrders++
	}

	if err := iter.Close(); err != nil {
		log.Printf("❌ Erreur lecture stats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lecture statistiques"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_orders": totalOrders,
		"total_revenue": totalRevenue,
		"by_status": stats,
	})
}
