package handlers

import (
	"cedra_back_end/internal/models"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v80"
	"github.com/stripe/stripe-go/v80/paymentintent"
	"github.com/stripe/stripe-go/v80/webhook"
)

func init() {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
}

// ✅ Crée un PaymentIntent Stripe
func CreatePaymentIntent(c *gin.Context) {
	var req struct {
		Items []models.CartItem `json:"items"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Requête invalide"})
		return
	}

	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Panier vide"})
		return
	}

	// 💰 Calcul du total
	var total float64
	for _, item := range req.Items {
		total += item.Price * float64(item.Quantity)
	}

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(int64(total * 100)), // en centimes
		Currency: stripe.String("eur"),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}

	intent, err := paymentintent.New(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"clientSecret": intent.ClientSecret,
	})
}

//
// 🔔 Stripe Webhook : confirmation du paiement
//
func StripeWebhook(c *gin.Context) {
	const MaxBodyBytes = int64(65536)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxBodyBytes)

	payload, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lecture corps échouée"})
		return
	}

	endpointSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")

	event, err := webhook.ConstructEvent(payload,
		c.GetHeader("Stripe-Signature"),
		endpointSecret,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Signature invalide"})
		return
	}

	switch event.Type {
	case "payment_intent.succeeded":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err == nil {
			// ✅ Paiement réussi
			log.Printf("✅ Paiement confirmé : %s (%.2f€)",
				pi.ID, float64(pi.Amount)/100)
		}

	case "payment_intent.payment_failed":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err == nil {
			log.Printf("⚠️ Paiement échoué : %s", pi.ID)
		}
	}

	c.Status(http.StatusOK)
}
