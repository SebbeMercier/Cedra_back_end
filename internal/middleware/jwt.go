package middleware

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

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

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("méthode de signature inattendue: %v", token.Header["alg"])
			}
			return jwtSecret, nil
		})

		if err != nil {
			log.Printf("❌ Erreur parsing JWT: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalide"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			log.Printf("✅ Claims JWT: %+v", claims)

			// Vérifier l'expiration
			if exp, ok := claims["exp"].(float64); ok {
				if time.Now().Unix() > int64(exp) {
					log.Println("❌ Token expiré")
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Token expiré"})
					c.Abort()
					return
				}
			}

			userID, ok := claims["user_id"].(string)
			if !ok {
				log.Printf("❌ user_id manquant ou invalide dans claims: %+v", claims)
				c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id manquant"})
				c.Abort()
				return
			}

			log.Printf("✅ user_id extrait: %s", userID)

			// ✅ Mettre les claims dans le context Gin
			c.Set("user_id", userID)
			c.Set("email", claims["email"])
			c.Set("role", claims["role"])

			// ✅ AJOUT CRITIQUE : Extraire isCompanyAdmin et le mettre dans le context
			if isCompanyAdmin, ok := claims["isCompanyAdmin"].(bool); ok {
				c.Set("isCompanyAdmin", isCompanyAdmin)
				log.Printf("✅ isCompanyAdmin extrait et mis dans context: %v", isCompanyAdmin)
			} else {
				// Par défaut false si absent
				c.Set("isCompanyAdmin", false)
				log.Printf("⚠️ isCompanyAdmin non trouvé dans claims, défini à false par défaut")
			}

			c.Next()
		} else {
			log.Println("❌ Claims invalides")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalide"})
			c.Abort()
			return
		}
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
