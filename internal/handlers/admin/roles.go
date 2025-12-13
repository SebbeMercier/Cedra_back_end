package admin

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"

	"cedra_back_end/internal/database"
	"cedra_back_end/internal/middleware"
	"cedra_back_end/internal/models"
	"cedra_back_end/internal/utils"
)

// GetAllRoles récupère tous les rôles
func GetAllRoles(c *gin.Context) {
	usersSession, err := database.GetUsersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	query := `SELECT id, name, display_name, description, permissions, is_active, created_at, updated_at FROM roles`
	iter := usersSession.Query(query).Iter()
	defer iter.Close()

	var roles []models.Role
	var role models.Role

	for iter.Scan(&role.ID, &role.Name, &role.DisplayName, &role.Description,
		&role.Permissions, &role.IsActive, &role.CreatedAt, &role.UpdatedAt) {
		roles = append(roles, role)
	}

	if err := iter.Close(); err != nil {
		log.Printf("❌ Erreur récupération rôles: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur serveur"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"roles": roles,
		"total": len(roles),
	})
}

// CreateRole crée un nouveau rôle
func CreateRole(c *gin.Context) {
	var req struct {
		Name        string   `json:"name" binding:"required"`
		DisplayName string   `json:"display_name" binding:"required"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides: " + err.Error()})
		return
	}

	usersSession, err := database.GetUsersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	// Vérifier si le nom existe déjà
	var existingName string
	checkQuery := `SELECT name FROM roles WHERE name = ? LIMIT 1`
	if err := usersSession.Query(checkQuery, req.Name).Scan(&existingName); err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Ce nom de rôle existe déjà"})
		return
	}

	role := models.Role{
		ID:          gocql.TimeUUID(),
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Permissions: req.Permissions,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	insertQuery := `
		INSERT INTO roles (id, name, display_name, description, permissions, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	if err := usersSession.Query(insertQuery,
		role.ID, role.Name, role.DisplayName, role.Description,
		role.Permissions, role.IsActive, role.CreatedAt, role.UpdatedAt,
	).Exec(); err != nil {
		log.Printf("❌ Erreur création rôle: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création du rôle"})
		return
	}

	// Log d'audit
	utils.LogAction(c, utils.ACTION_ROLE_CREATE, utils.RESOURCE_ROLE, role.ID.String(), nil, role)

	log.Printf("✅ Rôle créé: %s", role.Name)
	c.JSON(http.StatusCreated, gin.H{
		"message": "Rôle créé avec succès",
		"role":    role,
	})
}

// AssignRoleToUser assigne un rôle à un utilisateur
func AssignRoleToUser(c *gin.Context) {
	var req struct {
		UserID    string     `json:"user_id" binding:"required"`
		RoleID    string     `json:"role_id" binding:"required"`
		ExpiresAt *time.Time `json:"expires_at"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides: " + err.Error()})
		return
	}

	usersSession, err := database.GetUsersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	roleID, err := gocql.ParseUUID(req.RoleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID rôle invalide"})
		return
	}

	// Vérifier que le rôle existe
	var roleName string
	roleQuery := `SELECT name FROM roles WHERE id = ? AND is_active = true`
	if err := usersSession.Query(roleQuery, roleID).Scan(&roleName); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rôle non trouvé"})
		return
	}

	// Vérifier si l'utilisateur a déjà ce rôle
	var existingID gocql.UUID
	checkQuery := `SELECT id FROM user_roles WHERE user_id = ? AND role_id = ? AND is_active = true`
	if err := usersSession.Query(checkQuery, req.UserID, roleID).Scan(&existingID); err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "L'utilisateur a déjà ce rôle"})
		return
	}

	grantedBy, _ := c.Get("user_id")
	userRole := models.UserRole{
		ID:        gocql.TimeUUID(),
		UserID:    req.UserID,
		RoleID:    roleID,
		RoleName:  roleName,
		GrantedBy: grantedBy.(string),
		GrantedAt: time.Now(),
		ExpiresAt: req.ExpiresAt,
		IsActive:  true,
	}

	insertQuery := `
		INSERT INTO user_roles (id, user_id, role_id, role_name, granted_by, granted_at, expires_at, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	if err := usersSession.Query(insertQuery,
		userRole.ID, userRole.UserID, userRole.RoleID, userRole.RoleName,
		userRole.GrantedBy, userRole.GrantedAt, userRole.ExpiresAt, userRole.IsActive,
	).Exec(); err != nil {
		log.Printf("❌ Erreur attribution rôle: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de l'attribution du rôle"})
		return
	}

	// Log d'audit
	utils.LogAction(c, utils.ACTION_ROLE_ASSIGN, utils.RESOURCE_USER, req.UserID, nil, userRole)

	log.Printf("✅ Rôle %s attribué à l'utilisateur %s", roleName, req.UserID)
	c.JSON(http.StatusOK, gin.H{
		"message":   "Rôle attribué avec succès",
		"user_role": userRole,
	})
}

// RevokeRoleFromUser révoque un rôle d'un utilisateur
func RevokeRoleFromUser(c *gin.Context) {
	userID := c.Param("user_id")
	roleIDStr := c.Param("role_id")

	roleID, err := gocql.ParseUUID(roleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID rôle invalide"})
		return
	}

	usersSession, err := database.GetUsersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	// Récupérer les infos avant suppression pour l'audit
	var userRole models.UserRole
	selectQuery := `SELECT id, role_name FROM user_roles WHERE user_id = ? AND role_id = ? AND is_active = true`
	if err := usersSession.Query(selectQuery, userID, roleID).Scan(&userRole.ID, &userRole.RoleName); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Attribution de rôle non trouvée"})
		return
	}

	// Désactiver le rôle
	updateQuery := `UPDATE user_roles SET is_active = false WHERE user_id = ? AND role_id = ?`
	if err := usersSession.Query(updateQuery, userID, roleID).Exec(); err != nil {
		log.Printf("❌ Erreur révocation rôle: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la révocation du rôle"})
		return
	}

	// Log d'audit
	utils.LogAction(c, utils.ACTION_ROLE_REVOKE, utils.RESOURCE_USER, userID, userRole, nil)

	log.Printf("✅ Rôle %s révoqué pour l'utilisateur %s", userRole.RoleName, userID)
	c.JSON(http.StatusOK, gin.H{"message": "Rôle révoqué avec succès"})
}

// GetUserRoles récupère les rôles d'un utilisateur
func GetUserRoles(c *gin.Context) {
	userID := c.Param("user_id")

	usersSession, err := database.GetUsersSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	query := `SELECT id, role_id, role_name, granted_by, granted_at, expires_at, is_active 
			  FROM user_roles WHERE user_id = ?`
	iter := usersSession.Query(query, userID).Iter()
	defer iter.Close()

	var userRoles []models.UserRole
	var userRole models.UserRole

	for iter.Scan(&userRole.ID, &userRole.RoleID, &userRole.RoleName,
		&userRole.GrantedBy, &userRole.GrantedAt, &userRole.ExpiresAt, &userRole.IsActive) {
		userRole.UserID = userID
		userRoles = append(userRoles, userRole)
	}

	if err := iter.Close(); err != nil {
		log.Printf("❌ Erreur récupération rôles utilisateur: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur serveur"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"roles":   userRoles,
		"total":   len(userRoles),
	})
}

// GetMyPermissions récupère les permissions de l'utilisateur connecté
func GetMyPermissions(c *gin.Context) {
	userID, _ := c.Get("user_id")

	permissions, err := middleware.GetUserPermissions(userID.(string))
	if err != nil {
		log.Printf("❌ Erreur récupération permissions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur serveur"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":     userID,
		"permissions": permissions,
		"total":       len(permissions),
	})
}
