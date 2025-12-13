package admin

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
)

// GetAuditLogs récupère les logs d'audit avec filtres
func GetAuditLogs(c *gin.Context) {
	usersSession, err := database.GetUsersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	// Paramètres de filtrage
	userID := c.Query("user_id")
	action := c.Query("action")
	resource := c.Query("resource")
	success := c.Query("success")
	limitStr := c.DefaultQuery("limit", "100")

	limit, _ := strconv.Atoi(limitStr)
	if limit > 500 {
		limit = 500
	}

	// Construire la requête dynamiquement
	var query string
	var args []interface{}

	baseQuery := `SELECT id, user_id, user_email, action, resource, resource_id, 
				  old_value, new_value, ip_address, user_agent, success, 
				  error_msg, timestamp, session_id FROM audit_logs`

	conditions := []string{}

	if userID != "" {
		conditions = append(conditions, "user_id = ?")
		args = append(args, userID)
	}

	if action != "" {
		conditions = append(conditions, "action = ?")
		args = append(args, action)
	}

	if resource != "" {
		conditions = append(conditions, "resource = ?")
		args = append(args, resource)
	}

	if success != "" {
		successBool, _ := strconv.ParseBool(success)
		conditions = append(conditions, "success = ?")
		args = append(args, successBool)
	}

	if len(conditions) > 0 {
		query = baseQuery + " WHERE " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			query += " AND " + conditions[i]
		}
	} else {
		query = baseQuery
	}

	query += " LIMIT ?"
	args = append(args, limit)

	iter := usersSession.Query(query, args...).Iter()
	defer iter.Close()

	var logs []models.AuditLog
	var auditLog models.AuditLog

	for iter.Scan(&auditLog.ID, &auditLog.UserID, &auditLog.UserEmail,
		&auditLog.Action, &auditLog.Resource, &auditLog.ResourceID,
		&auditLog.OldValue, &auditLog.NewValue, &auditLog.IPAddress,
		&auditLog.UserAgent, &auditLog.Success, &auditLog.ErrorMsg,
		&auditLog.Timestamp, &auditLog.SessionID) {
		logs = append(logs, auditLog)
	}

	if err := iter.Close(); err != nil {
		log.Printf("❌ Erreur récupération logs audit: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur serveur"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"total": len(logs),
		"filters": gin.H{
			"user_id":  userID,
			"action":   action,
			"resource": resource,
			"success":  success,
			"limit":    limit,
		},
	})
}

// GetAuditLogsByResource récupère les logs pour une ressource spécifique
func GetAuditLogsByResource(c *gin.Context) {
	resource := c.Param("resource")
	resourceID := c.Param("resource_id")
	limitStr := c.DefaultQuery("limit", "50")

	limit, _ := strconv.Atoi(limitStr)
	if limit > 200 {
		limit = 200
	}

	usersSession, err := database.GetUsersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	query := `SELECT id, user_id, user_email, action, resource, resource_id, 
			  old_value, new_value, ip_address, user_agent, success, 
			  error_msg, timestamp, session_id FROM audit_logs 
			  WHERE resource = ? AND resource_id = ? LIMIT ?`

	iter := usersSession.Query(query, resource, resourceID, limit).Iter()
	defer iter.Close()

	var logs []models.AuditLog
	var auditLog models.AuditLog

	for iter.Scan(&auditLog.ID, &auditLog.UserID, &auditLog.UserEmail,
		&auditLog.Action, &auditLog.Resource, &auditLog.ResourceID,
		&auditLog.OldValue, &auditLog.NewValue, &auditLog.IPAddress,
		&auditLog.UserAgent, &auditLog.Success, &auditLog.ErrorMsg,
		&auditLog.Timestamp, &auditLog.SessionID) {
		logs = append(logs, auditLog)
	}

	if err := iter.Close(); err != nil {
		log.Printf("❌ Erreur récupération logs audit: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur serveur"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"resource":    resource,
		"resource_id": resourceID,
		"logs":        logs,
		"total":       len(logs),
	})
}

// GetAuditStats récupère les statistiques des logs d'audit
func GetAuditStats(c *gin.Context) {
	usersSession, err := database.GetUsersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	// Statistiques générales
	var totalLogs int
	if err := usersSession.Query(`SELECT COUNT(*) FROM audit_logs`).Scan(&totalLogs); err != nil {
		log.Printf("❌ Erreur comptage logs: %v", err)
		totalLogs = 0
	}

	var successfulActions int
	if err := usersSession.Query(`SELECT COUNT(*) FROM audit_logs WHERE success = true`).Scan(&successfulActions); err != nil {
		log.Printf("❌ Erreur comptage actions réussies: %v", err)
		successfulActions = 0
	}

	var failedActions int
	if err := usersSession.Query(`SELECT COUNT(*) FROM audit_logs WHERE success = false`).Scan(&failedActions); err != nil {
		log.Printf("❌ Erreur comptage actions échouées: %v", err)
		failedActions = 0
	}

	// Actions récentes (dernières 24h)
	yesterday := time.Now().Add(-24 * time.Hour)
	var recentActions int
	if err := usersSession.Query(`SELECT COUNT(*) FROM audit_logs WHERE timestamp > ?`, yesterday).Scan(&recentActions); err != nil {
		log.Printf("❌ Erreur comptage actions récentes: %v", err)
		recentActions = 0
	}

	// Top actions
	topActionsQuery := `SELECT action, COUNT(*) as count FROM audit_logs GROUP BY action LIMIT 10`
	iter := usersSession.Query(topActionsQuery).Iter()
	defer iter.Close()

	type ActionCount struct {
		Action string `json:"action"`
		Count  int    `json:"count"`
	}

	var topActions []ActionCount
	var action string
	var count int

	for iter.Scan(&action, &count) {
		topActions = append(topActions, ActionCount{
			Action: action,
			Count:  count,
		})
	}
	iter.Close()

	// Top utilisateurs
	topUsersQuery := `SELECT user_email, COUNT(*) as count FROM audit_logs WHERE user_email != '' GROUP BY user_email LIMIT 10`
	iter2 := usersSession.Query(topUsersQuery).Iter()
	defer iter2.Close()

	type UserCount struct {
		UserEmail string `json:"user_email"`
		Count     int    `json:"count"`
	}

	var topUsers []UserCount
	var userEmail string

	for iter2.Scan(&userEmail, &count) {
		topUsers = append(topUsers, UserCount{
			UserEmail: userEmail,
			Count:     count,
		})
	}
	iter2.Close()

	c.JSON(http.StatusOK, gin.H{
		"total_logs":         totalLogs,
		"successful_actions": successfulActions,
		"failed_actions":     failedActions,
		"recent_actions":     recentActions,
		"success_rate":       float64(successfulActions) / float64(totalLogs) * 100,
		"top_actions":        topActions,
		"top_users":          topUsers,
	})
}
