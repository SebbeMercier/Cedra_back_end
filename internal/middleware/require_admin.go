package middleware

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

// RequireAdmin vérifie que l'utilisateur a le rôle "admin"
func RequireAdmin(c *gin.Context) {
	role, exists := c.Get("role")
	if !exists || role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Accès réservé aux administrateurs"})
		c.Abort()
		return
	}
	c.Next()
}
