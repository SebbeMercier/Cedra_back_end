package payement

import (
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
	"cedra_back_end/internal/utils"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v83"
	"github.com/stripe/stripe-go/v83/paymentintent"
	"github.com/stripe/stripe-go/v83/webhook"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/bson"
)

// ✅ Crée un PaymentIntent Stripe
func CreatePaymentIntent(c *gin.Context) {
	var req struct {
		Items []models.CartItem `json:"items"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Requête invalide ou panier vide"})
		return
	}

	total := calcTotal(req.Items)
	userID := c.GetString("user_id")
	email := c.GetString("email")

	if userID == "" || email == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur non authentifié ou e-mail manquant"})
		return
	}

	// ✅ Sérialise le panier en JSON pour le stocker dans Stripe
	cartJSON, err := json.Marshal(req.Items)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur serialisation panier"})
		return
	}

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(int64(total * 100)),
		Currency: stripe.String("eur"),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
		Metadata: map[string]string{
			"user_id": userID,
			"email":   email,
			"cart":    string(cartJSON), // ✅ Sauvegarde le panier ici
		},
	}

	intent, err := paymentintent.New(params)
	if err != nil {
		log.Println("❌ Erreur Stripe:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("💳 PaymentIntent créé : %s (%.2f€) pour %s", intent.ID, total, email)

	c.JSON(http.StatusOK, gin.H{
		"clientSecret": intent.ClientSecret,
		"paymentId":    intent.ID,
	})
}

// ✅ Webhook Stripe
func StripeWebhook(c *gin.Context) {
	const MaxBodyBytes = int64(65536)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxBodyBytes)

	payload, err := c.GetRawData()
	if err != nil {
		log.Println("❌ Lecture payload échouée:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Échec lecture body"})
		return
	}

	secret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	var event stripe.Event

	if secret == "" {
		log.Println("⚠️ Pas de STRIPE_WEBHOOK_SECRET — mode test")
		if err := json.Unmarshal(payload, &event); err != nil {
			log.Println("❌ JSON invalide:", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "JSON invalide"})
			return
		}
	} else {
		event, err = webhook.ConstructEvent(payload, c.GetHeader("Stripe-Signature"), secret)
		if err != nil {
			log.Println("❌ Signature Stripe invalide:", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Signature invalide"})
			return
		}
	}

	log.Printf("📥 Événement Stripe reçu : %s", event.Type)
	handleStripeEvent(event)

	c.Status(http.StatusOK)
}

// ✅ Traitement de l’événement Stripe
func handleStripeEvent(event stripe.Event) {
	log.Println("✅ handleStripeEvent déclenché")

	if event.Type != "payment_intent.succeeded" {
		log.Printf("ℹ️ Événement ignoré : %s", event.Type)
		return
	}

	var pi stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		log.Println("❌ Erreur décodage PaymentIntent:", err)
		return
	}
	log.Printf("🧠 PaymentIntent reçu : %s", pi.ID)

	userID := pi.Metadata["user_id"]
	userEmail := pi.Metadata["email"]
	cartData := pi.Metadata["cart"] // ✅ Récupère depuis Stripe

	if userID == "" || userEmail == "" || cartData == "" {
		log.Println("⚠️ Métadonnées incomplètes")
		return
	}
	log.Printf("👤 User ID = %s | 📧 Email = %s", userID, userEmail)

	// Vérifier si la commande existe déjà
	ctx := context.Background()
	collection := database.MongoOrdersDB.Collection("orders")
	count, err := collection.CountDocuments(ctx, bson.M{"payment_intent_id": pi.ID})
	if err != nil {
		log.Println("❌ Erreur MongoDB count:", err)
		return
	}
	if count > 0 {
		log.Println("🔁 Commande déjà enregistrée, on ignore.")
		return
	}

	// ✅ Désérialise le panier depuis Stripe (pas depuis Redis)
	var cartItems []models.CartItem
	if err := json.Unmarshal([]byte(cartData), &cartItems); err != nil {
		log.Println("❌ Erreur JSON panier:", err)
		return
	}
	log.Printf("🛒 Articles dans le panier : %d", len(cartItems))

	// Créer la commande
	order := models.Order{
		ID:              primitive.NewObjectID(),
		UserID:          userID,
		PaymentIntentID: pi.ID,
		TotalPrice:      calcTotal(cartItems),
		Status:          "paid",
		CreatedAt:       time.Now(),
		Items:           []models.OrderItem{},
	}

	for _, item := range cartItems {
     order.Items = append(order.Items, models.OrderItem{
        ProductID: item.ProductID,
        Quantity:  item.Quantity,
        Price:     item.Price,
        Name:      item.Name, 
    })
	}

	log.Println("📤 Insertion commande MongoDB...")
	res, err := collection.InsertOne(ctx, order)
	if err != nil {
		log.Println("❌ Erreur insertion Mongo :", err)
		return
	}
	log.Printf("✅ Commande insérée avec ID = %v", res.InsertedID)

	// ✅ Supprimer le panier Redis APRÈS la commande
	key := "cart:" + userID
	if err := database.RedisClient.Del(ctx, key).Err(); err == nil {
		log.Printf("🧹 Panier supprimé Redis pour %s", userID)
	}

	// Générer l'HTML et le PDF, puis envoyer l'e-mail
	html := utils.GenerateOrderConfirmationHTML(order, userEmail)

	pdf, err := utils.GenerateInvoicePDF(order, userEmail)
	if err != nil {
		log.Println("❌ Erreur génération PDF :", err)
		pdf = nil
	}

	go func() {
		if err := utils.SendConfirmationEmail(userEmail, "Confirmation de votre commande Cedra", html, pdf); err != nil {
			log.Println("❌ Erreur envoi e-mail confirmation :", err)
			log.Printf("❌ Détails erreur : %+v", err)
		} else {
			log.Println("📧 E-mail de confirmation envoyé à", userEmail)
		}
	}()
}

func calcTotal(items []models.CartItem) float64 {
	var total float64
	for _, item := range items {
		total += item.Price * float64(item.Quantity)
	}
	return total
}