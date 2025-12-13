package utils

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"

	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
)

// LogAction enregistre une action dans les logs d'audit
func LogAction(c *gin.Context, action, resource string, resourceID string, oldValue, newValue interface{}) {
	go func() {
		if err := logActionAsync(c, action, resource, resourceID, oldValue, newValue, true, ""); err != nil {
			log.Printf("❌ Erreur enregistrement log audit: %v", err)
		}
	}()
}

// LogFailedAction enregistre une action échouée dans les logs d'audit
func LogFailedAction(c *gin.Context, action, resource, resourceID, errorMsg string) {
	go func() {
		if err := logActionAsync(c, action, resource, resourceID, nil, nil, false, errorMsg); err != nil {
			log.Printf("❌ Erreur enregistrement log audit: %v", err)
		}
	}()
}

// logActionAsync enregistre de façon asynchrone
func logActionAsync(c *gin.Context, action, resource, resourceID string, oldValue, newValue interface{}, success bool, errorMsg string) error {
	usersSession, err := database.GetUsersSession()
	if err != nil {
		return err
	}

	userID, _ := c.Get("user_id")
	userEmail, _ := c.Get("email")

	// Sérialiser les valeurs
	var oldValueStr, newValueStr string
	if oldValue != nil {
		if oldBytes, err := json.Marshal(oldValue); err == nil {
			oldValueStr = string(oldBytes)
		}
	}
	if newValue != nil {
		if newBytes, err := json.Marshal(newValue); err == nil {
			newValueStr = string(newBytes)
		}
	}

	auditLog := models.AuditLog{
		ID:         gocql.TimeUUID(),
		UserID:     getStringValue(userID),
		UserEmail:  getStringValue(userEmail),
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		OldValue:   oldValueStr,
		NewValue:   newValueStr,
		IPAddress:  c.ClientIP(),
		UserAgent:  c.GetHeader("User-Agent"),
		Success:    success,
		ErrorMsg:   errorMsg,
		Timestamp:  time.Now(),
		SessionID:  c.GetHeader("X-Session-ID"),
	}

	query := `
		INSERT INTO audit_logs (
			id, user_id, user_email, action, resource, resource_id,
			old_value, new_value, ip_address, user_agent, success,
			error_msg, timestamp, session_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	return usersSession.Query(query,
		auditLog.ID, auditLog.UserID, auditLog.UserEmail, auditLog.Action,
		auditLog.Resource, auditLog.ResourceID, auditLog.OldValue, auditLog.NewValue,
		auditLog.IPAddress, auditLog.UserAgent, auditLog.Success, auditLog.ErrorMsg,
		auditLog.Timestamp, auditLog.SessionID,
	).Exec()
}

// getStringValue convertit une interface{} en string
func getStringValue(value interface{}) string {
	if value == nil {
		return ""
	}
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}

// Actions d'audit prédéfinies
const (
	// Actions produits
	ACTION_PRODUCT_CREATE       = "product.create"
	ACTION_PRODUCT_UPDATE       = "product.update"
	ACTION_PRODUCT_DELETE       = "product.delete"
	ACTION_PRODUCT_PRICE_CHANGE = "product.price_change"

	// Actions commandes
	ACTION_ORDER_CREATE = "order.create"
	ACTION_ORDER_UPDATE = "order.update"
	ACTION_ORDER_CANCEL = "order.cancel"
	ACTION_ORDER_REFUND = "order.refund"

	// Actions utilisateurs
	ACTION_USER_CREATE = "user.create"
	ACTION_USER_UPDATE = "user.update"
	ACTION_USER_DELETE = "user.delete"
	ACTION_USER_BAN    = "user.ban"
	ACTION_USER_UNBAN  = "user.unban"

	// Actions coupons
	ACTION_COUPON_CREATE = "coupon.create"
	ACTION_COUPON_UPDATE = "coupon.update"
	ACTION_COUPON_DELETE = "coupon.delete"

	// Actions inventaire
	ACTION_STOCK_UPDATE = "stock.update"
	ACTION_STOCK_ALERT  = "stock.alert"

	// Actions rôles et permissions
	ACTION_ROLE_ASSIGN = "role.assign"
	ACTION_ROLE_REVOKE = "role.revoke"
	ACTION_ROLE_CREATE = "role.create"
	ACTION_ROLE_UPDATE = "role.update"
	ACTION_ROLE_DELETE = "role.delete"

	// Actions système
	ACTION_LOGIN_SUCCESS   = "auth.login_success"
	ACTION_LOGIN_FAILED    = "auth.login_failed"
	ACTION_LOGOUT          = "auth.logout"
	ACTION_SETTINGS_UPDATE = "settings.update"
)

// Resources d'audit
const (
	RESOURCE_PRODUCT   = "product"
	RESOURCE_ORDER     = "order"
	RESOURCE_USER      = "user"
	RESOURCE_COUPON    = "coupon"
	RESOURCE_INVENTORY = "inventory"
	RESOURCE_ROLE      = "role"
	RESOURCE_SETTINGS  = "settings"
	RESOURCE_AUTH      = "auth"
)
