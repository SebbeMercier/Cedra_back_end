package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CompanyAdminRequired vérifie que l'utilisateur est admin de sa société
func CompanyAdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		isCompanyAdmin, exists := c.Get("isCompanyAdmin")

		log.Printf("🔍 CompanyAdminRequired check: exists=%v, value=%v", exists, isCompanyAdmin)

		if !exists {
			log.Println("❌ isCompanyAdmin non trouvé dans le context")
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Accès réservé aux administrateurs de société",
			})
			c.Abort()
			return
		}

		// ✅ Vérification de type sécurisée
		adminBool, ok := isCompanyAdmin.(bool)
		if !ok {
			log.Printf("❌ isCompanyAdmin n'est pas un bool: type=%T, value=%v", isCompanyAdmin, isCompanyAdmin)
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Accès réservé aux administrateurs de société",
			})
			c.Abort()
			return
		}

		if !adminBool {
			log.Println("❌ isCompanyAdmin = false, accès refusé")
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Accès réservé aux administrateurs de société",
			})
			c.Abort()
			return
		}

		log.Println("✅ Utilisateur est bien admin de société, accès autorisé")
		c.Next()
	}
}
