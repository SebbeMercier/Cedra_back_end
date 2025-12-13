package pa

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"

	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
)

// CreateCoupon - Créer un nouveau coupon (Admin seulement)
func CreateCoupon(c *gin.Context) {
	var req struct {
		Code            string    `json:"code" binding:"required"`
		Type            string    `json:"type" binding:"required"` // "percentage", "fixed", "free_shipping"
		Value           float64   `json:"value" binding:"required"`
		MinAmount       float64   `json:"min_amount"`
		MaxAmount       *float64  `json:"max_amount"`
		MaxUses         int       `json:"max_uses"`
		MaxUsesPerUser  int       `json:"max_uses_per_user"`
		ApplicableToAll bool      `json:"applicable_to_all"`
		ProductIDs      []string  `json:"product_ids"`
		CategoryIDs     []string  `json:"category_ids"`
		ExpiresAt       time.Time `json:"expires_at" binding:"required"`
		StartsAt        time.Time `json:"starts_at"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides: " + err.Error()})
		return
	}

	// Validation du type
	if req.Type != "percentage" && req.Type != "fixed" && req.Type != "free_shipping" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Type de coupon invalide"})
		return
	}

	// Validation des valeurs
	if req.Type == "percentage" && (req.Value <= 0 || req.Value > 100) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pourcentage doit être entre 1 et 100"})
		return
	}

	if req.Type == "fixed" && req.Value <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Montant fixe doit être positif"})
		return
	}

	// Vérifier si le code existe déjà
	ordersSession, err := database.GetOrdersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	var existingCode string
	query := `SELECT code FROM ks_orders.coupons WHERE code = ? LIMIT 1`
	if err := ordersSession.Query(query, strings.ToUpper(req.Code)).Scan(&existingCode); err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Ce code coupon existe déjà"})
		return
	}

	userID, _ := c.Get("user_id")
	couponID := gocql.TimeUUID()
	now := time.Now()

	if req.StartsAt.IsZero() {
		req.StartsAt = now
	}

	coupon := models.Coupon{
		ID:              couponID,
		Code:            strings.ToUpper(req.Code),
		Type:            req.Type,
		Value:           req.Value,
		MinAmount:       req.MinAmount,
		MaxAmount:       req.MaxAmount,
		MaxUses:         req.MaxUses,
		UsedCount:       0,
		MaxUsesPerUser:  req.MaxUsesPerUser,
		ApplicableToAll: req.ApplicableToAll,
		ProductIDs:      req.ProductIDs,
		CategoryIDs:     req.CategoryIDs,
		ExpiresAt:       req.ExpiresAt,
		StartsAt:        req.StartsAt,
		IsActive:        true,
		CreatedBy:       userID.(string),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Insérer le coupon
	insertQuery := `
		INSERT INTO ks_orders.coupons (
			id, code, type, value, min_amount, max_amount, max_uses, used_count,
			max_uses_per_user, applicable_to_all, product_ids, category_ids,
			expires_at, starts_at, is_active, created_by, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	if err := ordersSession.Query(insertQuery,
		coupon.ID, coupon.Code, coupon.Type, coupon.Value, coupon.MinAmount,
		coupon.MaxAmount, coupon.MaxUses, coupon.UsedCount, coupon.MaxUsesPerUser,
		coupon.ApplicableToAll, coupon.ProductIDs, coupon.CategoryIDs,
		coupon.ExpiresAt, coupon.StartsAt, coupon.IsActive, coupon.CreatedBy,
		coupon.CreatedAt, coupon.UpdatedAt,
	).Exec(); err != nil {
		log.Printf("❌ Erreur création coupon: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création du coupon"})
		return
	}

	log.Printf("✅ Coupon créé: %s", coupon.Code)
	c.JSON(http.StatusCreated, gin.H{
		"message": "Coupon créé avec succès",
		"coupon":  coupon,
	})
}

// ValidateCouponDetailed - Valider un coupon avec détails
func ValidateCouponDetailed(c *gin.Context) {
	code := c.Query("code")
	cartTotalStr := c.Query("cart_total")
	userID, _ := c.Get("user_id")

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Code coupon requis"})
		return
	}

	cartTotal, err := strconv.ParseFloat(cartTotalStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Montant du panier invalide"})
		return
	}

	validation := validateCoupon(code, cartTotal, userID.(string))
	c.JSON(http.StatusOK, validation)
}

// GetAllCoupons - Récupérer tous les coupons (Admin)
func GetAllCoupons(c *gin.Context) {
	query := `SELECT id, code, type, value, min_amount, max_amount, max_uses, used_count,
			  max_uses_per_user, applicable_to_all, expires_at, starts_at, is_active,
			  created_by, created_at, updated_at FROM ks_orders.coupons`

	ordersSession, err := database.GetOrdersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	iter := ordersSession.Query(query).Iter()
	defer iter.Close()

	var coupons []models.Coupon
	var coupon models.Coupon

	for iter.Scan(&coupon.ID, &coupon.Code, &coupon.Type, &coupon.Value,
		&coupon.MinAmount, &coupon.MaxAmount, &coupon.MaxUses, &coupon.UsedCount,
		&coupon.MaxUsesPerUser, &coupon.ApplicableToAll, &coupon.ExpiresAt,
		&coupon.StartsAt, &coupon.IsActive, &coupon.CreatedBy, &coupon.CreatedAt,
		&coupon.UpdatedAt) {
		coupons = append(coupons, coupon)
	}

	if err := iter.Close(); err != nil {
		log.Printf("❌ Erreur récupération coupons: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur serveur"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"coupons": coupons,
		"total":   len(coupons),
	})
}

// UpdateCoupon - Mettre à jour un coupon
func UpdateCoupon(c *gin.Context) {
	couponID := c.Param("id")

	var req struct {
		IsActive  *bool      `json:"is_active"`
		MaxUses   *int       `json:"max_uses"`
		ExpiresAt *time.Time `json:"expires_at"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides"})
		return
	}

	id, err := gocql.ParseUUID(couponID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID coupon invalide"})
		return
	}

	updates := []string{}
	values := []interface{}{}

	if req.IsActive != nil {
		updates = append(updates, "is_active = ?")
		values = append(values, *req.IsActive)
	}

	if req.MaxUses != nil {
		updates = append(updates, "max_uses = ?")
		values = append(values, *req.MaxUses)
	}

	if req.ExpiresAt != nil {
		updates = append(updates, "expires_at = ?")
		values = append(values, *req.ExpiresAt)
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Aucune mise à jour fournie"})
		return
	}

	updates = append(updates, "updated_at = ?")
	values = append(values, time.Now())
	values = append(values, id)

	query := fmt.Sprintf("UPDATE ks_orders.coupons SET %s WHERE id = ?", strings.Join(updates, ", "))

	ordersSession, err := database.GetOrdersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	if err := ordersSession.Query(query, values...).Exec(); err != nil {
		log.Printf("❌ Erreur mise à jour coupon: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Coupon mis à jour avec succès"})
}

// DeleteCoupon - Supprimer un coupon
func DeleteCoupon(c *gin.Context) {
	couponID := c.Param("id")

	id, err := gocql.ParseUUID(couponID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID coupon invalide"})
		return
	}

	query := `DELETE FROM ks_orders.coupons WHERE id = ?`
	ordersSession, err := database.GetOrdersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	if err := ordersSession.Query(query, id).Exec(); err != nil {
		log.Printf("❌ Erreur suppression coupon: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Coupon supprimé avec succès"})
}

// validateCoupon - Fonction utilitaire pour valider un coupon
func validateCoupon(code string, cartTotal float64, userID string) models.CouponValidation {
	// Récupérer le coupon
	var coupon models.Coupon
	query := `SELECT id, code, type, value, min_amount, max_amount, max_uses, used_count,
			  max_uses_per_user, applicable_to_all, expires_at, starts_at, is_active
			  FROM ks_orders.coupons WHERE code = ? LIMIT 1`

	ordersSession, err := database.GetOrdersSession()
	if err != nil {
		return models.CouponValidation{
			IsValid:      false,
			ErrorMessage: "Erreur serveur",
		}
	}

	if err := ordersSession.Query(query, strings.ToUpper(code)).Scan(
		&coupon.ID, &coupon.Code, &coupon.Type, &coupon.Value, &coupon.MinAmount,
		&coupon.MaxAmount, &coupon.MaxUses, &coupon.UsedCount, &coupon.MaxUsesPerUser,
		&coupon.ApplicableToAll, &coupon.ExpiresAt, &coupon.StartsAt, &coupon.IsActive,
	); err != nil {
		return models.CouponValidation{
			IsValid:      false,
			ErrorMessage: "Code coupon invalide",
		}
	}

	// Vérifications
	now := time.Now()

	if !coupon.IsActive {
		return models.CouponValidation{
			IsValid:      false,
			ErrorMessage: "Ce coupon n'est plus actif",
		}
	}

	if now.Before(coupon.StartsAt) {
		return models.CouponValidation{
			IsValid:      false,
			ErrorMessage: "Ce coupon n'est pas encore valide",
		}
	}

	if now.After(coupon.ExpiresAt) {
		return models.CouponValidation{
			IsValid:      false,
			ErrorMessage: "Ce coupon a expiré",
		}
	}

	if coupon.MaxUses > 0 && coupon.UsedCount >= coupon.MaxUses {
		return models.CouponValidation{
			IsValid:      false,
			ErrorMessage: "Ce coupon a atteint sa limite d'utilisation",
		}
	}

	if cartTotal < coupon.MinAmount {
		return models.CouponValidation{
			IsValid:      false,
			ErrorMessage: fmt.Sprintf("Montant minimum requis: %.2f€", coupon.MinAmount),
		}
	}

	// Vérifier utilisation par utilisateur
	if coupon.MaxUsesPerUser > 0 {
		var userUsageCount int
		userUsageQuery := `SELECT COUNT(*) FROM ks_orders.coupon_usage WHERE coupon_id = ? AND user_id = ?`
		if err := ordersSession.Query(userUsageQuery, coupon.ID, userID).Scan(&userUsageCount); err == nil {
			if userUsageCount >= coupon.MaxUsesPerUser {
				return models.CouponValidation{
					IsValid:      false,
					ErrorMessage: "Vous avez déjà utilisé ce coupon le nombre maximum de fois",
				}
			}
		}
	}

	// Calculer la réduction
	var discount float64
	switch coupon.Type {
	case "percentage":
		discount = cartTotal * (coupon.Value / 100)
		if coupon.MaxAmount != nil && discount > *coupon.MaxAmount {
			discount = *coupon.MaxAmount
		}
	case "fixed":
		discount = coupon.Value
		if discount > cartTotal {
			discount = cartTotal
		}
	case "free_shipping":
		discount = 0 // Géré séparément dans le checkout
	}

	return models.CouponValidation{
		IsValid:  true,
		Discount: discount,
		Type:     coupon.Type,
		Code:     coupon.Code,
	}
}
