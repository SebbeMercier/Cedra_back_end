package pa

import (
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v83"
	"github.com/stripe/stripe-go/v83/paymentintent"
	"github.com/stripe/stripe-go/v83/promotioncode"
)

// Checkout crée une commande complète avec validation stock et coupons
func Checkout(c *gin.Context) {
	var req struct {
		AddressID  string `json:"address_id" binding:"required"`
		CouponCode string `json:"coupon_code"` // Optionnel
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides", "details": err.Error()})
		return
	}

	userID := c.GetString("user_id")
	email := c.GetString("email")

	if userID == "" || email == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur non authentifié"})
		return
	}

	// ✅ 1. Récupérer le panier depuis Redis
	ctx := context.Background()
	cartKey := "cart:" + userID

	cartData, err := database.Redis.Get(ctx, cartKey).Result()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Panier vide ou introuvable"})
		return
	}

	var cartItems []models.CartItem
	if err := json.Unmarshal([]byte(cartData), &cartItems); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lecture panier"})
		return
	}

	if len(cartItems) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Panier vide"})
		return
	}

	// ✅ 2. Vérifier l'adresse existe et appartient à l'utilisateur
	addressUUID, err := uuid.Parse(req.AddressID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID adresse invalide"})
		return
	}

	usersSession, err := database.GetUsersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	var addressUserID string
	err = usersSession.Query("SELECT user_id FROM addresses WHERE address_id = ?", gocql.UUID(addressUUID)).Scan(&addressUserID)
	if err != nil || addressUserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Adresse introuvable ou non autorisée"})
		return
	}

	// ✅ 3. Vérifier le stock pour chaque produit
	productsSession, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	for i, item := range cartItems {
		productUUID, err := uuid.Parse(item.ProductID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID produit invalide: " + item.ProductID})
			return
		}

		var stock int
		var name string
		var price float64
		err = productsSession.Query("SELECT stock, name, price FROM products WHERE product_id = ?", gocql.UUID(productUUID)).
			Scan(&stock, &name, &price)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Produit introuvable: " + item.ProductID})
			return
		}

		// Vérifier stock suffisant
		if stock < item.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":     "Stock insuffisant",
				"product":   name,
				"available": stock,
				"requested": item.Quantity,
			})
			return
		}

		// Mettre à jour les infos du panier avec les données actuelles
		cartItems[i].Name = name
		cartItems[i].Price = price
	}

	// ✅ 4. Calculer le total
	totalPrice := calcTotal(cartItems)

	// ✅ 5. Valider et appliquer le coupon (si fourni)
	var discountAmount float64
	var couponCode string
	var couponType string

	if req.CouponCode != "" {
		validation := validateCoupon(req.CouponCode, totalPrice, userID)
		if !validation.IsValid {
			c.JSON(http.StatusBadRequest, gin.H{"error": validation.ErrorMessage})
			return
		}

		discountAmount = validation.Discount
		couponCode = validation.Code
		couponType = validation.Type
		log.Printf("✅ Coupon appliqué: %s (%.2f€ de réduction)", couponCode, discountAmount)
	}

	finalPrice := totalPrice - discountAmount
	if finalPrice < 0 {
		finalPrice = 0
	}

	// ✅ 6. Sérialiser le panier pour Stripe metadata
	cartJSON, err := json.Marshal(cartItems)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur sérialisation panier"})
		return
	}

	// ✅ 7. Créer le PaymentIntent Stripe
	metadata := map[string]string{
		"user_id":    userID,
		"email":      email,
		"address_id": req.AddressID,
		"cart":       string(cartJSON),
	}

	// Ajouter le coupon dans les métadonnées si présent
	if couponCode != "" {
		metadata["coupon_code"] = couponCode
		metadata["coupon_type"] = couponType
		metadata["discount_amount"] = string(rune(int(discountAmount * 100)))
	}

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(int64(finalPrice * 100)),
		Currency: stripe.String("eur"),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
		Metadata: metadata,
	}

	intent, err := paymentintent.New(params)
	if err != nil {
		log.Printf("❌ Erreur Stripe: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur création paiement", "details": err.Error()})
		return
	}

	log.Printf("💳 Checkout créé: %s (%.2f€ → %.2f€) pour %s", intent.ID, totalPrice, finalPrice, email)

	// ✅ 8. Réponse avec détails
	c.JSON(http.StatusOK, gin.H{
		"client_secret":   intent.ClientSecret,
		"payment_id":      intent.ID,
		"amount":          finalPrice,
		"original_amount": totalPrice,
		"discount":        discountAmount,
		"currency":        "eur",
		"items_count":     len(cartItems),
	})
}

// ValidateCoupon vérifie si un code promo est valide
func ValidateCoupon(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Code requis"})
		return
	}

	// Rechercher le promotion code dans Stripe
	params := &stripe.PromotionCodeListParams{}
	params.Filters.AddFilter("code", "", code)
	params.Filters.AddFilter("active", "", "true")

	iter := promotioncode.List(params)
	if !iter.Next() {
		c.JSON(http.StatusNotFound, gin.H{
			"valid": false,
			"error": "Code invalide ou expiré",
		})
		return
	}

	promo := iter.PromotionCode()

	response := gin.H{
		"valid":  true,
		"code":   code,
		"active": promo.Active,
		"id":     promo.ID,
	}

	// Informations supplémentaires
	if promo.ExpiresAt > 0 {
		response["expires_at"] = promo.ExpiresAt
	}
	if promo.MaxRedemptions > 0 {
		response["max_redemptions"] = promo.MaxRedemptions
		response["times_redeemed"] = promo.TimesRedeemed
	}

	c.JSON(http.StatusOK, response)
}
