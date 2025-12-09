package pa

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
	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v83"
	"github.com/stripe/stripe-go/v83/paymentintent"
	"github.com/stripe/stripe-go/v83/webhook"
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
	session, err := database.GetOrdersSession()
	if err != nil {
		log.Printf("❌ Erreur session ScyllaDB: %v", err)
		return
	}

	// Vérifier si une commande avec ce payment_intent_id existe déjà
	var existingOrderID gocql.UUID
	err = session.Query("SELECT order_id FROM orders WHERE payment_intent_id = ? ALLOW FILTERING", pi.ID).Scan(&existingOrderID)
	if err == nil {
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

	// Créer les items de commande
	var orderItems []models.OrderItem
	for _, item := range cartItems {
		orderItems = append(orderItems, models.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     item.Price,
			Name:      item.Name,
		})
	}

	// Sérialiser les items en JSON pour ScyllaDB
	itemsJSON, err := json.Marshal(orderItems)
	if err != nil {
		log.Printf("❌ Erreur sérialisation items: %v", err)
		return
	}

	// Créer la commande
	orderID := gocql.TimeUUID()
	now := time.Now()
	totalPrice := calcTotal(cartItems)

	log.Println("📤 Insertion commande ScyllaDB...")

	// Insert dans orders
	err = session.Query(`INSERT INTO orders (order_id, user_id, payment_intent_id, items, total_price, status, created_at, updated_at) 
	                     VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		orderID, userID, pi.ID, string(itemsJSON), totalPrice, "paid", now, now).Exec()
	if err != nil {
		log.Printf("❌ Erreur insertion ScyllaDB : %v", err)
		return
	}

	// Insert dans orders_by_user pour l'index
	err = session.Query(`INSERT INTO orders_by_user (user_id, order_id, payment_intent_id, items, total_price, status, created_at) 
	                     VALUES (?, ?, ?, ?, ?, ?, ?)`,
		userID, orderID, pi.ID, string(itemsJSON), totalPrice, "paid", now).Exec()
	if err != nil {
		log.Printf("⚠️ Erreur insertion index orders_by_user: %v", err)
	}

	log.Printf("✅ Commande insérée avec ID = %s", orderID.String())

	// ✅ Décrémenter le stock pour chaque produit
	if err := decrementStock(orderItems); err != nil {
		log.Printf("⚠️ Erreur décrémentation stock: %v", err)
	} else {
		log.Println("✅ Stock décrémenté avec succès")
	}

	// Créer l'objet order pour les fonctions utils
	order := models.Order{
		ID:              orderID,
		UserID:          userID,
		PaymentIntentID: pi.ID,
		TotalPrice:      totalPrice,
		Status:          "paid",
		CreatedAt:       now,
		Items:           orderItems,
	}

	// ✅ Supprimer le panier Redis APRÈS la commande
	ctx := context.Background()
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

// decrementStock décrémente le stock des produits après un paiement réussi
func decrementStock(orderItems []models.OrderItem) error {
	productsSession, err := database.GetProductsSession()
	if err != nil {
		return err
	}

	for _, item := range orderItems {
		productUUID, parseErr := uuid.Parse(item.ProductID)
		if parseErr != nil {
			log.Printf("⚠️ ID produit invalide: %s", item.ProductID)
			continue
		}

		// Décrémenter le stock
		execErr := productsSession.Query(
			"UPDATE products SET stock = stock - ? WHERE product_id = ?",
			item.Quantity,
			gocql.UUID(productUUID),
		).Exec()

		if execErr != nil {
			log.Printf("❌ Erreur décrémentation stock pour %s: %v", item.ProductID, execErr)
			return execErr
		}

		log.Printf("📦 Stock décrémenté: %s (-%d)", item.Name, item.Quantity)
	}

	return nil
}
