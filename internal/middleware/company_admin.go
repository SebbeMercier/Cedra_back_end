package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CompanyAdminRequired vérifie que l'utilisateur est admin de sa société
func CompanyAdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		isCompanyAdmin, exists := c.Get("isCompanyAdmin")
		
		if !exists || !isCompanyAdmin.(bool) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Accès réservé aux administrateurs de société",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}