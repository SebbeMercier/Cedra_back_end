package pa

import (
	"cedra_back_end/internal/database"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GetDashboardStats retourne les statistiques du dashboard admin
func GetDashboardStats(c *gin.Context) {
	session, err := database.GetOrdersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	// Statistiques des commandes
	var totalOrders int
	var totalRevenue float64
	statusCount := make(map[string]int)

	iter := session.Query("SELECT status, total_price FROM orders").Iter()
	var status string
	var price float64

	for iter.Scan(&status, &price) {
		totalOrders++
		totalRevenue += price
		statusCount[status]++
	}

	if err := iter.Close(); err != nil {
		log.Printf("❌ Erreur lecture stats: %v", err)
	}

	// Statistiques des produits
	productsSession, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	var totalProducts int
	var lowStockProducts int
	var outOfStockProducts int

	prodIter := productsSession.Query("SELECT stock FROM products").Iter()
	var stock int

	for prodIter.Scan(&stock) {
		totalProducts++
		if stock == 0 {
			outOfStockProducts++
		} else if stock < 10 {
			lowStockProducts++
		}
	}

	if err := prodIter.Close(); err != nil {
		log.Printf("❌ Erreur lecture produits: %v", err)
	}

	// Statistiques des utilisateurs
	usersSession, err := database.GetUsersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	var totalUsers int
	usersIter := usersSession.Query("SELECT user_id FROM users").Iter()
	var userID string

	for usersIter.Scan(&userID) {
		totalUsers++
	}

	if err := usersIter.Close(); err != nil {
		log.Printf("❌ Erreur lecture utilisateurs: %v", err)
	}

	// Calculer les moyennes
	var averageOrderValue float64
	if totalOrders > 0 {
		averageOrderValue = totalRevenue / float64(totalOrders)
	}

	// Statistiques des remboursements
	var totalRefunds int
	var pendingRefunds int

	refundsIter := session.Query("SELECT status FROM refunds").Iter()
	var refundStatus string

	for refundsIter.Scan(&refundStatus) {
		totalRefunds++
		if refundStatus == "pending" {
			pendingRefunds++
		}
	}

	if err := refundsIter.Close(); err != nil {
		log.Printf("❌ Erreur lecture remboursements: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": gin.H{
			"total":               totalOrders,
			"total_revenue":       totalRevenue,
			"average_order_value": averageOrderValue,
			"by_status":           statusCount,
		},
		"products": gin.H{
			"total":        totalProducts,
			"low_stock":    lowStockProducts,
			"out_of_stock": outOfStockProducts,
		},
		"users": gin.H{
			"total": totalUsers,
		},
		"refunds": gin.H{
			"total":   totalRefunds,
			"pending": pendingRefunds,
		},
		"generated_at": time.Now(),
	})
}

// GetRecentOrders retourne les commandes récentes
func GetRecentOrders(c *gin.Context) {
	limit := 10 // Par défaut 10 commandes

	session, err := database.GetOrdersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	iter := session.Query(`
		SELECT order_id, user_id, payment_intent_id, total_price, status, created_at
		FROM orders LIMIT ?
	`, limit).Iter()

	type RecentOrder struct {
		ID              string    `json:"id"`
		UserID          string    `json:"user_id"`
		PaymentIntentID string    `json:"payment_intent_id"`
		TotalPrice      float64   `json:"total_price"`
		Status          string    `json:"status"`
		CreatedAt       time.Time `json:"created_at"`
	}

	var orders []RecentOrder
	var order RecentOrder
	var orderID interface{}

	for iter.Scan(&orderID, &order.UserID, &order.PaymentIntentID, &order.TotalPrice, &order.Status, &order.CreatedAt) {
		order.ID = orderID.(string)
		orders = append(orders, order)
	}

	if err := iter.Close(); err != nil {
		log.Printf("❌ Erreur lecture commandes récentes: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lecture commandes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"count":  len(orders),
	})
}

// GetTopProducts retourne les produits les plus vendus
func GetTopProducts(c *gin.Context) {
	limit := 10

	session, err := database.GetOrdersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	// Compter les ventes par produit
	productSales := make(map[string]int)

	iter := session.Query("SELECT items FROM orders").Iter()
	var itemsJSON string

	for iter.Scan(&itemsJSON) {
		// Parser le JSON pour compter les produits
		// Note: Simplification - en production, utiliser un vrai parser JSON
		if len(itemsJSON) > 0 {
			// Compter approximativement (à améliorer)
			productSales["product_placeholder"]++
		}
	}

	if err := iter.Close(); err != nil {
		log.Printf("❌ Erreur lecture top produits: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Top produits (à implémenter avec parsing JSON complet)",
		"limit":   limit,
	})
}
