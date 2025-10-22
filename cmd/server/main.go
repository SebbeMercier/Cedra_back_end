package main

import (
	"cedra_back_end/internal/config"
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/routes"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v83"
)

func main() {
	// 🔹 1. Charger le .env
	config.Load()

	secret := os.Getenv("STRIPE_SECRET_KEY")
	if secret == "" {
		log.Println("❌ STRIPE_SECRET_KEY absente")
	} else {
		log.Println("✅ STRIPE_SECRET_KEY présente")
	}

	// ✅ Initialiser Stripe ici (après .env)
	stripe.Key = secret
	if stripe.Key == "" {
		log.Fatal("❌ Impossible d’initialiser Stripe : clé manquante")
	} else {
		log.Println("✅ Stripe initialisé avec la clé secrète")
	}

	// 🔹 2. Connexion aux bases
	database.ConnectDatabases()

	// 🔹 3. Créer le serveur
	r := gin.Default()
	routes.RegisterRoutes(r)

	// 🔹 4. Lancer le serveur
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("🚀 Serveur démarré sur le port", port)
	r.Run(":" + port)
}
