package middleware

import (
	"bytes"
	"cedra_back_end/internal/database"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// Limites par endpoint
	LoginMaxAttempts          = 5
	RegisterMaxAttempts       = 3
	ForgotPasswordMaxAttempts = 3
	APIMaxRequests            = 100 // Par minute pour les endpoints généraux

	// Durées de cooldown
	LoginCooldown          = 15 * time.Minute
	RegisterCooldown       = 30 * time.Minute
	ForgotPasswordCooldown = 10 * time.Minute
	APICooldown            = 1 * time.Minute
)

// LoginRateLimit limite les tentatives de connexion par email
func LoginRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Lire le body sans le consommer
		bodyBytes, _ := ioutil.ReadAll(c.Request.Body)

		var input struct {
			Email string `json:"email"`
		}

		if err := json.Unmarshal(bodyBytes, &input); err != nil || input.Email == "" {
			c.Next()
			return
		}

		// Remettre le body pour les handlers suivants
		c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

		ctx := context.Background()
		key := "login_attempts:" + input.Email

		// Vérifier si l'utilisateur est en cooldown
		cooldownKey := "login_cooldown:" + input.Email
		if database.Redis.Exists(ctx, cooldownKey).Val() > 0 {
			ttl := database.Redis.TTL(ctx, cooldownKey).Val()
			minutes := int(ttl.Minutes())
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       fmt.Sprintf("Trop de tentatives échouées. Réessayez dans %d minutes", minutes),
				"retry_after": int(ttl.Seconds()),
			})
			c.Abort()
			return
		}

		// Vérifier le nombre de tentatives
		attempts, _ := database.Redis.Get(ctx, key).Int()
		if attempts >= LoginMaxAttempts {
			// Activer le cooldown
			database.Redis.Set(ctx, cooldownKey, "1", LoginCooldown)
			database.Redis.Del(ctx, key)

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       fmt.Sprintf("Trop de tentatives échouées. Compte bloqué pendant %d minutes", int(LoginCooldown.Minutes())),
				"retry_after": int(LoginCooldown.Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()

		// Si login échoué (401), incrémenter les tentatives
		if c.Writer.Status() == http.StatusUnauthorized {
			database.Redis.Incr(ctx, key)
			database.Redis.Expire(ctx, key, LoginCooldown)

			remaining := LoginMaxAttempts - attempts - 1
			if remaining > 0 {
				c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			}
		} else if c.Writer.Status() == http.StatusOK {
			// Login réussi, réinitialiser les tentatives
			database.Redis.Del(ctx, key)
			database.Redis.Del(ctx, cooldownKey)
		}
	}
}

// RegisterRateLimit limite les inscriptions par IP
func RegisterRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()
		ip := c.ClientIP()
		key := "register_attempts:" + ip

		// Vérifier si l'IP est en cooldown
		cooldownKey := "register_cooldown:" + ip
		if database.Redis.Exists(ctx, cooldownKey).Val() > 0 {
			ttl := database.Redis.TTL(ctx, cooldownKey).Val()
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       fmt.Sprintf("Trop d'inscriptions. Réessayez dans %d minutes", int(ttl.Minutes())),
				"retry_after": int(ttl.Seconds()),
			})
			c.Abort()
			return
		}

		// Vérifier le nombre de tentatives
		attempts, _ := database.Redis.Get(ctx, key).Int()
		if attempts >= RegisterMaxAttempts {
			// Activer le cooldown
			database.Redis.Set(ctx, cooldownKey, "1", RegisterCooldown)
			database.Redis.Del(ctx, key)

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       fmt.Sprintf("Trop d'inscriptions. Réessayez dans %d minutes", int(RegisterCooldown.Minutes())),
				"retry_after": int(RegisterCooldown.Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()

		// Si inscription réussie, incrémenter
		if c.Writer.Status() == http.StatusCreated {
			database.Redis.Incr(ctx, key)
			database.Redis.Expire(ctx, key, RegisterCooldown)
		}
	}
}

// ForgotPasswordRateLimit limite les demandes de reset de mot de passe
func ForgotPasswordRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Email string `json:"email"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email requis"})
			c.Abort()
			return
		}

		ctx := context.Background()
		key := "forgot_password_attempts:" + input.Email

		// Vérifier si l'email est en cooldown
		cooldownKey := "forgot_password_cooldown:" + input.Email
		if database.Redis.Exists(ctx, cooldownKey).Val() > 0 {
			ttl := database.Redis.TTL(ctx, cooldownKey).Val()
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       fmt.Sprintf("Trop de demandes. Réessayez dans %d minutes", int(ttl.Minutes())),
				"retry_after": int(ttl.Seconds()),
			})
			c.Abort()
			return
		}

		// Vérifier le nombre de tentatives
		attempts, _ := database.Redis.Get(ctx, key).Int()
		if attempts >= ForgotPasswordMaxAttempts {
			// Activer le cooldown
			database.Redis.Set(ctx, cooldownKey, "1", ForgotPasswordCooldown)
			database.Redis.Del(ctx, key)

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       fmt.Sprintf("Trop de demandes. Réessayez dans %d minutes", int(ForgotPasswordCooldown.Minutes())),
				"retry_after": int(ForgotPasswordCooldown.Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()

		// Incrémenter après chaque demande
		if c.Writer.Status() == http.StatusOK {
			database.Redis.Incr(ctx, key)
			database.Redis.Expire(ctx, key, ForgotPasswordCooldown)
		}
	}
}

// APIRateLimit limite le nombre de requêtes par IP (général)
func APIRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()
		ip := c.ClientIP()
		key := "api_requests:" + ip

		// Vérifier le nombre de requêtes dans la dernière minute
		requests, _ := database.Redis.Get(ctx, key).Int()
		if requests >= APIMaxRequests {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Trop de requêtes. Réessayez dans 1 minute",
				"retry_after": 60,
			})
			c.Abort()
			return
		}

		// Incrémenter le compteur
		pipe := database.Redis.Pipeline()
		pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, APICooldown)
		pipe.Exec(ctx)

		// Ajouter les headers de rate limit
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", APIMaxRequests))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", APIMaxRequests-requests-1))

		c.Next()
	}
}

// CartRateLimit limite les ajouts au panier (anti-spam)
func CartRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		if userID == "" {
			c.Next()
			return
		}

		ctx := context.Background()
		key := "cart_add:" + userID

		// Max 20 ajouts par minute
		requests, _ := database.Redis.Get(ctx, key).Int()
		if requests >= 20 {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Trop d'ajouts au panier. Ralentissez un peu",
				"retry_after": 60,
			})
			c.Abort()
			return
		}

		// Incrémenter
		pipe := database.Redis.Pipeline()
		pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, 1*time.Minute)
		pipe.Exec(ctx)

		c.Next()
	}
}

// SearchRateLimit limite les recherches (anti-spam)
func SearchRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		ctx := context.Background()
		key := "search_requests:" + ip

		// Max 30 recherches par minute
		requests, _ := database.Redis.Get(ctx, key).Int()
		if requests >= 30 {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Trop de recherches. Réessayez dans 1 minute",
				"retry_after": 60,
			})
			c.Abort()
			return
		}

		// Incrémenter
		pipe := database.Redis.Pipeline()
		pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, 1*time.Minute)
		pipe.Exec(ctx)

		c.Next()
	}
}
