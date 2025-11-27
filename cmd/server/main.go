package main

import (
	"cedra_back_end/internal/config"
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/routes"
	"context"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/facebook"
	"github.com/markbates/goth/providers/google"
	"github.com/stripe/stripe-go/v83"
)

func main() {
	config.Load()

	secret := os.Getenv("STRIPE_SECRET_KEY")
	stripe.Key = secret
	if stripe.Key == "" {
		log.Fatal("❌ Impossible d'initialiser Stripe : clé manquante")
	}
	log.Println("✅ Stripe initialisé")

	database.ConnectDatabases()

	// ✅ Initialiser les prepared statements pour améliorer les performances
	database.InitPreparedStatements()

	// ✅ Pré-chauffer le cache Redis
	warmupRedisCache()

	initOAuthProviders()

	r := gin.Default()
	routes.RegisterRoutes(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("🚀 Serveur Cedra lancé sur le port", port)
	r.Run(":" + port)
}

func initOAuthProviders() {
	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		log.Fatal("❌ SESSION_SECRET manquant dans .env")
	}

	// ✅ Configuration du store
	store := sessions.NewCookieStore([]byte(sessionSecret))
	store.MaxAge(86400 * 30)
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 30,
		HttpOnly: true,
		Secure:   false, // false en dev, true en prod
		SameSite: http.SameSiteLaxMode,
	}

	gothic.Store = store

	// ✅ CRITIQUE : Fonction pour extraire le provider depuis l'URL
	gothic.GetProviderName = func(req *http.Request) (string, error) {
		// Essaye d'abord les query params
		if provider := req.URL.Query().Get("provider"); provider != "" {
			return provider, nil
		}

		// Ensuite essaye le form
		if provider := req.FormValue("provider"); provider != "" {
			return provider, nil
		}

		// ✅ FIX : Retourne une erreur générique
		return "", errors.New("provider not found")
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	googleCallback := baseURL + "/api/auth/google/callback"
	facebookCallback := baseURL + "/api/auth/facebook/callback"

	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	facebookClientID := os.Getenv("FACEBOOK_CLIENT_ID")
	facebookClientSecret := os.Getenv("FACEBOOK_CLIENT_SECRET")

	var providers []goth.Provider

	if googleClientID != "" && googleClientSecret != "" {
		providers = append(providers, google.New(
			googleClientID,
			googleClientSecret,
			googleCallback,
		))
		log.Println("✅ Google OAuth activé")
	}

	if facebookClientID != "" && facebookClientSecret != "" {
		providers = append(providers, facebook.New(
			facebookClientID,
			facebookClientSecret,
			facebookCallback,
		))
		log.Println("✅ Facebook OAuth activé")
	}

	if len(providers) == 0 {
		log.Println("⚠️ Aucun provider OAuth configuré")
		return
	}

	goth.UseProviders(providers...)
	log.Printf("✅ %d OAuth provider(s) initialisé(s)", len(providers))
}

// warmupRedisCache pré-chauffe le cache Redis pour éviter la latence du premier appel
func warmupRedisCache() {
	ctx := context.Background()
	// Faire un ping pour établir la connexion
	if err := database.Redis.Ping(ctx).Err(); err == nil {
		log.Println("✅ Cache Redis pré-chauffé")
	}
}
