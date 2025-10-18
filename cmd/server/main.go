package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/apple"
	"github.com/markbates/goth/providers/facebook"
	"github.com/markbates/goth/providers/google"
	"github.com/gin-contrib/cors"
	"cedra_back_end/internal/config"
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/routes"
	"time"
)

func main() {
	// 1️⃣ Charger la configuration (fichier .env)
	config.Load()

	database.ConnectDatabases()

	goth.UseProviders(
		google.New(
			os.Getenv("GOOGLE_CLIENT_ID"),
			os.Getenv("GOOGLE_CLIENT_SECRET"),
			"http://localhost:8080/auth/google/callback",
			"email", "profile",
		),
		facebook.New(
			os.Getenv("FACEBOOK_CLIENT_ID"),
			os.Getenv("FACEBOOK_CLIENT_SECRET"),
			"http://localhost:8080/auth/facebook/callback",
			"email",
		),
		apple.New(
			os.Getenv("APPLE_CLIENT_ID"),
			os.Getenv("APPLE_TEAM_ID"),
			os.Getenv("APPLE_KEY_ID"),
			http.DefaultClient,
			os.Getenv("APPLE_PRIVATE_KEY"),
			"http://localhost:8080/auth/apple/callback",
			"email", "name",
		),
	)

	r := gin.Default()

	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		log.Fatal("❌ SESSION_SECRET manquant dans .env")
	}
	store := cookie.NewStore([]byte(secret))
	r.Use(sessions.Sessions("auth-session", store))


	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // ou "http://192.168.1.200"
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	routes.RegisterRoutes(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Serveur Cedra démarré sur le port %s\n", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("❌ Erreur serveur : %v", err)
	}
}
