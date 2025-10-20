package main

import (
	"cedra_back_end/internal/config"
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/routes"
	"github.com/gin-gonic/gin"
	"log"
	"os"
)

func main() {
	// 🔹 1. Charger le .env
	config.Load()

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
