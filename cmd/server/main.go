package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/facebook"
	"github.com/markbates/goth/providers/google"

	"cedra_back_end/internal/config"
	"cedra_back_end/internal/database"
)

type ctxKey string

const providerKey ctxKey = "provider"

func main() {
	config.Load()
	database.Connect()

	// Init Goth providers
	goth.UseProviders(
		google.New(
			os.Getenv("GOOGLE_CLIENT_ID"),
			os.Getenv("GOOGLE_CLIENT_SECRET"),
			"http://localhost:8080/auth/google/callback",
			"email", "profile"),
		facebook.New(
			os.Getenv("FACEBOOK_CLIENT_ID"),
			os.Getenv("FACEBOOK_CLIENT_SECRET"),
			"http://localhost:8080/auth/facebook/callback",
			"email"),
	)

	// Gin router
	r := gin.Default()

	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		log.Fatal("❌ SESSION_SECRET manquant dans .env")
	}
	store := cookie.NewStore([]byte(secret))

	r.Use(sessions.Sessions("auth-session", store))

	// Routes OAuth
	r.GET("/auth/:provider", beginAuth)
	r.GET("/auth/:provider/callback", callbackAuth)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Serveur démarré sur :%s\n", port)
	http.ListenAndServe(":"+port, r)
}

// Handlers OAuth
func beginAuth(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "aucun provider spécifié"})
		return
	}

	// Injecte provider dans le contexte pour gothic
	c.Request = c.Request.WithContext(
		context.WithValue(c.Request.Context(), providerKey, provider),
	)

	gothic.BeginAuthHandler(c.Writer, c.Request)
}

func callbackAuth(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "aucun provider spécifié"})
		return
	}

	c.Request = c.Request.WithContext(
		context.WithValue(c.Request.Context(), providerKey, provider),
	)

	user, err := gothic.CompleteUserAuth(c.Writer, c.Request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"provider": user.Provider,
		"email":    user.Email,
		"user_id":  user.UserID,
	})
}
