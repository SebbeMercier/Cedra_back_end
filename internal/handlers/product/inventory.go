package product

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"

	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
)

// UpdateStock - Mettre à jour le stock d'un produit
func UpdateStock(c *gin.Context) {
	productIDStr := c.Param("id")

	var req struct {
		Quantity int    `json:"quantity" binding:"required"`
		Reason   string `json:"reason" binding:"required"`
		Type     string `json:"type" binding:"required"` // "restock", "adjustment"
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides: " + err.Error()})
		return
	}

	productID, err := gocql.ParseUUID(productIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID produit invalide"})
		return
	}

	// Récupérer le stock actuel
	var currentStock int
	var productName string

	productsSession, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	query := `SELECT stock, name FROM ks_products.products WHERE product_id = ?`
	if err := productsSession.Query(query, productID).Scan(&currentStock, &productName); err != nil {
		log.Printf("❌ Produit non trouvé: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Produit non trouvé"})
		return
	}

	var newStock int
	switch req.Type {
	case "restock":
		newStock = currentStock + req.Quantity
	case "adjustment":
		newStock = req.Quantity // Quantité absolue
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Type d'opération invalide"})
		return
	}

	if newStock < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le stock ne peut pas être négatif"})
		return
	}

	userID, _ := c.Get("user_id")

	// Mettre à jour le stock
	updateQuery := `UPDATE ks_products.products SET stock = ?, updated_at = ? WHERE product_id = ?`
	if err := productsSession.Query(updateQuery, newStock, time.Now(), productID).Exec(); err != nil {
		log.Printf("❌ Erreur mise à jour stock: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour du stock"})
		return
	}

	// Enregistrer le mouvement de stock
	movement := models.StockMovement{
		ID:        gocql.TimeUUID(),
		ProductID: productID,
		Type:      req.Type,
		Quantity:  req.Quantity,
		PrevStock: currentStock,
		NewStock:  newStock,
		Reason:    req.Reason,
		UserID:    userID.(string),
		CreatedAt: time.Now(),
	}

	insertMovementQuery := `
		INSERT INTO ks_products.stock_movements (
			id, product_id, type, quantity, prev_stock, new_stock, reason, user_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	if err := productsSession.Query(insertMovementQuery,
		movement.ID, movement.ProductID, movement.Type, movement.Quantity,
		movement.PrevStock, movement.NewStock, movement.Reason, movement.UserID,
		movement.CreatedAt,
	).Exec(); err != nil {
		log.Printf("⚠️ Erreur enregistrement mouvement stock: %v", err)
	}

	// Vérifier les alertes de stock faible
	checkLowStockAlert(productID, productName, newStock)

	log.Printf("✅ Stock mis à jour pour %s: %d -> %d", productName, currentStock, newStock)
	c.JSON(http.StatusOK, gin.H{
		"message":     "Stock mis à jour avec succès",
		"prev_stock":  currentStock,
		"new_stock":   newStock,
		"movement_id": movement.ID,
	})
}

// GetStockMovements - Récupérer l'historique des mouvements de stock
func GetStockMovements(c *gin.Context) {
	productIDStr := c.Query("product_id")
	limitStr := c.DefaultQuery("limit", "50")

	limit, _ := strconv.Atoi(limitStr)
	if limit > 100 {
		limit = 100
	}

	var query string
	var args []interface{}

	if productIDStr != "" {
		productID, err := gocql.ParseUUID(productIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID produit invalide"})
			return
		}
		query = `SELECT id, product_id, type, quantity, prev_stock, new_stock, reason, user_id, created_at 
				 FROM ks_products.stock_movements WHERE product_id = ? LIMIT ?`
		args = []interface{}{productID, limit}
	} else {
		query = `SELECT id, product_id, type, quantity, prev_stock, new_stock, reason, user_id, created_at 
				 FROM ks_products.stock_movements LIMIT ?`
		args = []interface{}{limit}
	}

	productsSession, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	iter := productsSession.Query(query, args...).Iter()
	defer iter.Close()

	var movements []models.StockMovement
	var movement models.StockMovement

	for iter.Scan(&movement.ID, &movement.ProductID, &movement.Type, &movement.Quantity,
		&movement.PrevStock, &movement.NewStock, &movement.Reason, &movement.UserID,
		&movement.CreatedAt) {
		movements = append(movements, movement)
	}

	if err := iter.Close(); err != nil {
		log.Printf("❌ Erreur récupération mouvements: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur serveur"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"movements": movements,
		"total":     len(movements),
	})
}

// GetLowStockAlerts - Récupérer les alertes de stock faible
func GetLowStockAlerts(c *gin.Context) {
	productsSession, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	query := `SELECT id, product_id, product_name, current_stock, threshold_stock, alert_type, 
			  is_resolved, created_at FROM ks_products.stock_alerts WHERE is_resolved = false`

	iter := productsSession.Query(query).Iter()
	defer iter.Close()

	var alerts []models.StockAlert
	var alert models.StockAlert

	for iter.Scan(&alert.ID, &alert.ProductID, &alert.ProductName, &alert.CurrentStock,
		&alert.ThresholdStock, &alert.AlertType, &alert.IsResolved, &alert.CreatedAt) {
		alerts = append(alerts, alert)
	}

	if err := iter.Close(); err != nil {
		log.Printf("❌ Erreur récupération alertes: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur serveur"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts": alerts,
		"total":  len(alerts),
	})
}

// ResolveStockAlert - Marquer une alerte comme résolue
func ResolveStockAlert(c *gin.Context) {
	alertIDStr := c.Param("id")

	alertID, err := gocql.ParseUUID(alertIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID alerte invalide"})
		return
	}

	productsSession, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	now := time.Now()
	query := `UPDATE ks_products.stock_alerts SET is_resolved = true, resolved_at = ? WHERE id = ?`

	if err := productsSession.Query(query, now, alertID).Exec(); err != nil {
		log.Printf("❌ Erreur résolution alerte: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la résolution"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Alerte marquée comme résolue"})
}

// GetInventoryStats - Statistiques d'inventaire
func GetInventoryStats(c *gin.Context) {
	productsSession, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	var stats models.InventoryStats

	// Total des produits
	var totalProducts int
	if err := productsSession.Query(`SELECT COUNT(*) FROM ks_products.products WHERE is_active = true`).Scan(&totalProducts); err != nil {
		log.Printf("❌ Erreur comptage produits: %v", err)
		totalProducts = 0
	}
	stats.TotalProducts = totalProducts

	// Produits en stock faible
	var lowStockProducts int
	if err := productsSession.Query(`SELECT COUNT(*) FROM ks_products.products WHERE stock <= low_stock_threshold AND stock > 0 AND is_active = true`).Scan(&lowStockProducts); err != nil {
		log.Printf("❌ Erreur comptage stock faible: %v", err)
		lowStockProducts = 0
	}
	stats.LowStockProducts = lowStockProducts

	// Produits en rupture
	var outOfStockProducts int
	if err := productsSession.Query(`SELECT COUNT(*) FROM ks_products.products WHERE stock = 0 AND is_active = true`).Scan(&outOfStockProducts); err != nil {
		log.Printf("❌ Erreur comptage rupture: %v", err)
		outOfStockProducts = 0
	}
	stats.OutOfStockProducts = outOfStockProducts

	// Valeur totale de l'inventaire
	iter := productsSession.Query(`SELECT stock, price FROM ks_products.products WHERE is_active = true`).Iter()
	var totalValue float64
	var stock int
	var price float64

	for iter.Scan(&stock, &price) {
		totalValue += float64(stock) * price
	}
	iter.Close()
	stats.TotalValue = totalValue

	// Top produits vendus (simulé - nécessiterait une table de ventes)
	stats.TopSellingProducts = []models.ProductSales{}

	c.JSON(http.StatusOK, stats)
}

// checkLowStockAlert - Vérifier et créer des alertes de stock faible
func checkLowStockAlert(productID gocql.UUID, productName string, currentStock int) {
	// Récupérer le seuil de stock faible
	var threshold int
	query := `SELECT low_stock_threshold FROM ks_products.products WHERE product_id = ?`
	productsSession, err := database.GetProductsSession()
	if err != nil {
		return
	}

	if err := productsSession.Query(query, productID).Scan(&threshold); err != nil {
		return
	}

	if threshold == 0 {
		threshold = 10 // Seuil par défaut
	}

	var alertType string
	var shouldAlert bool

	if currentStock == 0 {
		alertType = "out_of_stock"
		shouldAlert = true
	} else if currentStock <= threshold {
		alertType = "low_stock"
		shouldAlert = true
	}

	if !shouldAlert {
		return
	}

	// Vérifier si une alerte non résolue existe déjà
	var existingAlertID gocql.UUID
	checkQuery := `SELECT id FROM ks_products.stock_alerts WHERE product_id = ? AND is_resolved = false LIMIT 1`
	if err := productsSession.Query(checkQuery, productID).Scan(&existingAlertID); err == nil {
		// Alerte existe déjà
		return
	}

	// Créer nouvelle alerte
	alert := models.StockAlert{
		ID:             gocql.TimeUUID(),
		ProductID:      productID,
		ProductName:    productName,
		CurrentStock:   currentStock,
		ThresholdStock: threshold,
		AlertType:      alertType,
		IsResolved:     false,
		CreatedAt:      time.Now(),
	}

	insertQuery := `
		INSERT INTO ks_products.stock_alerts (
			id, product_id, product_name, current_stock, threshold_stock, 
			alert_type, is_resolved, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	if err := productsSession.Query(insertQuery,
		alert.ID, alert.ProductID, alert.ProductName, alert.CurrentStock,
		alert.ThresholdStock, alert.AlertType, alert.IsResolved, alert.CreatedAt,
	).Exec(); err != nil {
		log.Printf("⚠️ Erreur création alerte stock: %v", err)
	} else {
		log.Printf("🚨 Alerte stock créée pour %s: %s", productName, alertType)
	}
}
