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

	"cedra_back_end/internal/config"
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/routes"
)

func main() {
	// 1️⃣ Charger la configuration (fichier .env)
	config.Load()

	// 2️⃣ Connexion à Mongo, Redis et Elasticsearch
	database.ConnectDatabases()

	// 3️⃣ Configuration des providers OAuth
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

	// 4️⃣ Initialisation du moteur Gin
	r := gin.Default()

	// 5️⃣ Configuration des sessions
	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		log.Fatal("❌ SESSION_SECRET manquant dans .env")
	}
	store := cookie.NewStore([]byte(secret))
	r.Use(sessions.Sessions("auth-session", store))

	// 6️⃣ Enregistrement des routes (ton package interne)
	routes.RegisterRoutes(r)

	// 7️⃣ Lancement du serveur HTTP
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Serveur Cedra démarré sur le port %s\n", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("❌ Erreur serveur : %v", err)
	}
}
