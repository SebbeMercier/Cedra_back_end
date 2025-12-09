package pa

import (
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v83"
	"github.com/stripe/stripe-go/v83/refund"
)

// RequestRefund permet à un utilisateur de demander un remboursement
func RequestRefund(c *gin.Context) {
	userID := c.GetString("user_id")
	orderID := c.Param("orderId")

	var req struct {
		Reason string `json:"reason" binding:"required,min=10,max=500"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides", "details": err.Error()})
		return
	}

	orderUUID, err := uuid.Parse(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID commande invalide"})
		return
	}

	// Vérifier que la commande existe et appartient à l'utilisateur
	session, err := database.GetOrdersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	var orderUserID string
	var paymentIntentID string
	var totalPrice float64
	var status string

	err = session.Query(`
		SELECT user_id, payment_intent_id, total_price, status
		FROM orders WHERE order_id = ?
	`, gocql.UUID(orderUUID)).Scan(&orderUserID, &paymentIntentID, &totalPrice, &status)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Commande introuvable"})
		return
	}

	if orderUserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cette commande ne vous appartient pas"})
		return
	}

	// Vérifier que la commande est éligible au remboursement
	if status != "paid" && status != "shipped" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cette commande n'est pas éligible au remboursement"})
		return
	}

	// Vérifier qu'il n'y a pas déjà une demande de remboursement
	var existingRefundID gocql.UUID
	err = session.Query("SELECT refund_id FROM refunds WHERE order_id = ? ALLOW FILTERING", gocql.UUID(orderUUID)).Scan(&existingRefundID)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Une demande de remboursement existe déjà pour cette commande"})
		return
	}

	// Créer la demande de remboursement
	refundID := gocql.TimeUUID()
	now := time.Now()

	err = session.Query(`
		INSERT INTO refunds (refund_id, order_id, user_id, reason, status, refund_amount, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, refundID, gocql.UUID(orderUUID), userID, req.Reason, "pending", totalPrice, now).Exec()

	if err != nil {
		log.Printf("❌ Erreur création demande remboursement: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur création demande"})
		return
	}

	log.Printf("💰 Demande de remboursement créée: %s pour commande %s", refundID, orderID)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Demande de remboursement créée",
		"refund": models.Refund{
			ID:           refundID,
			OrderID:      gocql.UUID(orderUUID),
			UserID:       userID,
			Reason:       req.Reason,
			Status:       "pending",
			RefundAmount: totalPrice,
			CreatedAt:    now,
		},
	})
}

// ProcessRefund traite une demande de remboursement (admin)
func ProcessRefund(c *gin.Context) {
	refundID := c.Param("refundId")

	var req struct {
		Action string `json:"action" binding:"required"` // approve, reject
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides"})
		return
	}

	if req.Action != "approve" && req.Action != "reject" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Action invalide (approve ou reject)"})
		return
	}

	refundUUID, err := uuid.Parse(refundID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID remboursement invalide"})
		return
	}

	session, err := database.GetOrdersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	// Récupérer les infos du remboursement
	var orderID gocql.UUID
	var refundAmount float64
	var refundStatus string

	err = session.Query(`
		SELECT order_id, refund_amount, status
		FROM refunds WHERE refund_id = ?
	`, gocql.UUID(refundUUID)).Scan(&orderID, &refundAmount, &refundStatus)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Demande de remboursement introuvable"})
		return
	}

	if refundStatus != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cette demande a déjà été traitée"})
		return
	}

	now := time.Now()

	if req.Action == "reject" {
		// Rejeter la demande
		err = session.Query(`
			UPDATE refunds SET status = ?, updated_at = ? WHERE refund_id = ?
		`, "rejected", now, gocql.UUID(refundUUID)).Exec()

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur mise à jour"})
			return
		}

		log.Printf("❌ Remboursement rejeté: %s", refundID)

		c.JSON(http.StatusOK, gin.H{
			"message": "Demande de remboursement rejetée",
			"status":  "rejected",
		})
		return
	}

	// Approuver et traiter le remboursement via Stripe
	var paymentIntentID string
	err = session.Query("SELECT payment_intent_id FROM orders WHERE order_id = ?", orderID).Scan(&paymentIntentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur récupération commande"})
		return
	}

	// Créer le remboursement Stripe
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(paymentIntentID),
		Amount:        stripe.Int64(int64(refundAmount * 100)),
		Reason:        stripe.String("requested_by_customer"),
	}

	stripeRefund, err := refund.New(params)
	if err != nil {
		log.Printf("❌ Erreur Stripe refund: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur traitement remboursement Stripe", "details": err.Error()})
		return
	}

	// Mettre à jour le statut
	err = session.Query(`
		UPDATE refunds SET status = ?, stripe_refund_id = ?, updated_at = ? WHERE refund_id = ?
	`, "completed", stripeRefund.ID, now, gocql.UUID(refundUUID)).Exec()

	if err != nil {
		log.Printf("⚠️ Erreur mise à jour refund: %v", err)
	}

	// Mettre à jour le statut de la commande
	session.Query("UPDATE orders SET status = ? WHERE order_id = ?", "refunded", orderID).Exec()

	log.Printf("✅ Remboursement traité: %s (Stripe: %s)", refundID, stripeRefund.ID)

	c.JSON(http.StatusOK, gin.H{
		"message":          "Remboursement traité avec succès",
		"status":           "completed",
		"stripe_refund_id": stripeRefund.ID,
		"amount":           refundAmount,
	})
}

// GetUserRefunds récupère les demandes de remboursement d'un utilisateur
func GetUserRefunds(c *gin.Context) {
	userID := c.GetString("user_id")

	session, err := database.GetOrdersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	iter := session.Query(`
		SELECT refund_id, order_id, reason, status, refund_amount, stripe_refund_id, created_at, updated_at
		FROM refunds WHERE user_id = ? ALLOW FILTERING
	`, userID).Iter()

	var refunds []models.Refund
	var refund models.Refund

	for iter.Scan(&refund.ID, &refund.OrderID, &refund.Reason, &refund.Status, &refund.RefundAmount, &refund.StripeRefundID, &refund.CreatedAt, &refund.UpdatedAt) {
		refund.UserID = userID
		refunds = append(refunds, refund)
	}

	if err := iter.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lecture remboursements"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"refunds": refunds,
		"count":   len(refunds),
	})
}

// GetAllRefunds récupère toutes les demandes de remboursement (admin)
func GetAllRefunds(c *gin.Context) {
	session, err := database.GetOrdersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	iter := session.Query(`
		SELECT refund_id, order_id, user_id, reason, status, refund_amount, stripe_refund_id, created_at, updated_at
		FROM refunds
	`).Iter()

	var refunds []models.Refund
	var refund models.Refund

	for iter.Scan(&refund.ID, &refund.OrderID, &refund.UserID, &refund.Reason, &refund.Status, &refund.RefundAmount, &refund.StripeRefundID, &refund.CreatedAt, &refund.UpdatedAt) {
		refunds = append(refunds, refund)
	}

	if err := iter.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lecture remboursements"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"refunds": refunds,
		"count":   len(refunds),
	})
}
