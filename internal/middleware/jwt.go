package middleware

import (
	"log"
	"net/http"
	"strings"

	"cedra_back_end/internal/cache"
	"cedra_back_end/internal/utils"

	"github.com/gin-gonic/gin"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		log.Printf("🔐 Authorization header reçu: %s", authHeader)

		if authHeader == "" {
			log.Println("❌ Pas de header Authorization")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token manquant"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			log.Printf("❌ Format Authorization invalide: %v parties", len(parts))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Format Authorization invalide"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		log.Printf("🎫 Token (20 premiers chars): %s...", tokenString[:min(20, len(tokenString))])

		// Parser le token avec les nouveaux claims
		claims, err := utils.ParseAccessToken(tokenString)
		if err != nil {
			log.Printf("❌ Erreur parsing JWT: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalide"})
			c.Abort()
			return
		}

		log.Printf("✅ Claims JWT: %+v", claims)

		// ✅ SÉCURITÉ 1: Vérifier si le token est blacklisté (révoqué)
		if cache.IsTokenBlacklisted(claims.TokenID) {
			log.Printf("❌ Token blacklisté (révoqué): %s", claims.TokenID)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token révoqué"})
			c.Abort()
			return
		}

		// ✅ SÉCURITÉ 2: Vérifier si l'utilisateur est banni
		if cache.IsUserBanned(claims.UserID) {
			log.Printf("❌ Utilisateur banni: %s", claims.UserID)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Compte banni"})
			c.Abort()
			return
		}

		log.Printf("✅ user_id extrait: %s", claims.UserID)

		// ✅ Mettre les claims dans le context Gin
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Set("isCompanyAdmin", claims.IsCompanyAdmin)
		c.Set("token_id", claims.TokenID) // Pour blacklist lors du logout

		log.Printf("✅ isCompanyAdmin: %v", claims.IsCompanyAdmin)

		c.Next()
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func AuthJWT(c *gin.Context) {
	AuthRequired()(c)
}
