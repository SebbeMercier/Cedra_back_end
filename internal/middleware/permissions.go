package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"cedra_back_end/internal/database"
)

// RequirePermission vérifie qu'un utilisateur a une permission spécifique
func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur non authentifié"})
			c.Abort()
			return
		}

		hasPermission, err := checkUserPermission(userID.(string), permission)
		if err != nil {
			log.Printf("❌ Erreur vérification permission: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur serveur"})
			c.Abort()
			return
		}

		if !hasPermission {
			log.Printf("🚫 Permission refusée: %s pour utilisateur %s", permission, userID)
			c.JSON(http.StatusForbidden, gin.H{
				"error":               "Permission insuffisante",
				"required_permission": permission,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyPermission vérifie qu'un utilisateur a au moins une des permissions
func RequireAnyPermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur non authentifié"})
			c.Abort()
			return
		}

		for _, permission := range permissions {
			hasPermission, err := checkUserPermission(userID.(string), permission)
			if err != nil {
				log.Printf("❌ Erreur vérification permission: %v", err)
				continue
			}

			if hasPermission {
				c.Next()
				return
			}
		}

		log.Printf("🚫 Aucune permission requise pour utilisateur %s: %v", userID, permissions)
		c.JSON(http.StatusForbidden, gin.H{
			"error":                "Permission insuffisante",
			"required_permissions": permissions,
		})
		c.Abort()
	}
}

// checkUserPermission vérifie si un utilisateur a une permission spécifique
func checkUserPermission(userID, permission string) (bool, error) {
	usersSession, err := database.GetUsersSession()
	if err != nil {
		return false, err
	}

	// Récupérer les rôles actifs de l'utilisateur
	query := `SELECT role_id FROM user_roles WHERE user_id = ? AND is_active = true`
	iter := usersSession.Query(query, userID).Iter()
	defer iter.Close()

	var roleIDs []string
	var roleID string
	for iter.Scan(&roleID) {
		roleIDs = append(roleIDs, roleID)
	}

	if err := iter.Close(); err != nil {
		return false, err
	}

	if len(roleIDs) == 0 {
		return false, nil
	}

	// Vérifier les permissions pour chaque rôle
	for _, roleID := range roleIDs {
		var permissions []string
		roleQuery := `SELECT permissions FROM roles WHERE id = ? AND is_active = true`
		if err := usersSession.Query(roleQuery, roleID).Scan(&permissions); err != nil {
			continue
		}

		// Vérifier si la permission est dans la liste
		for _, perm := range permissions {
			if perm == permission {
				return true, nil
			}
		}
	}

	return false, nil
}

// GetUserPermissions récupère toutes les permissions d'un utilisateur
func GetUserPermissions(userID string) ([]string, error) {
	usersSession, err := database.GetUsersSession()
	if err != nil {
		return nil, err
	}

	// Récupérer les rôles actifs de l'utilisateur
	query := `SELECT role_id FROM user_roles WHERE user_id = ? AND is_active = true`
	iter := usersSession.Query(query, userID).Iter()
	defer iter.Close()

	var roleIDs []string
	var roleID string
	for iter.Scan(&roleID) {
		roleIDs = append(roleIDs, roleID)
	}

	if err := iter.Close(); err != nil {
		return nil, err
	}

	// Collecter toutes les permissions
	permissionSet := make(map[string]bool)
	for _, roleID := range roleIDs {
		var permissions []string
		roleQuery := `SELECT permissions FROM roles WHERE id = ? AND is_active = true`
		if err := usersSession.Query(roleQuery, roleID).Scan(&permissions); err != nil {
			continue
		}

		for _, perm := range permissions {
			permissionSet[perm] = true
		}
	}

	// Convertir en slice
	var userPermissions []string
	for perm := range permissionSet {
		userPermissions = append(userPermissions, perm)
	}

	return userPermissions, nil
}

// HasPermission vérifie si l'utilisateur actuel a une permission (pour utilisation dans les handlers)
func HasPermission(c *gin.Context, permission string) bool {
	userID, exists := c.Get("user_id")
	if !exists {
		return false
	}

	hasPermission, err := checkUserPermission(userID.(string), permission)
	if err != nil {
		log.Printf("❌ Erreur vérification permission: %v", err)
		return false
	}

	return hasPermission
}
