package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/facebook"
	"github.com/markbates/goth/providers/google"
	"github.com/markbates/goth/providers/apple"

	"cedra_back_end/internal/config"
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/routes"
)

func main() {
	config.Load()
	database.ConnectMongo()
	log.Println("SESSION_SECRET:", os.Getenv("SESSION_SECRET"))


	// ✅ Enregistre les providers OAuth
	
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
		 apple.New(
			os.Getenv("APPLE_CLIENT_ID"),
			os.Getenv("APPLE_TEAM_ID"),
			os.Getenv("APPLE_KEY_ID"),
			http.DefaultClient,                            // ⬅️ Le *http.Client est maintenant en 4ème position
			os.Getenv("APPLE_PRIVATE_KEY"),              // ⬅️ La clé privée (string) est en 5ème position
			"http://localhost:8080/auth/apple/callback", // ⬅️ L'URL de rappel (string) est en 6ème position
			"email", "name"),
			)

	r := gin.Default()

	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		log.Fatal("❌ SESSION_SECRET manquant dans .env")
	}
	store := cookie.NewStore([]byte(secret))
	r.Use(sessions.Sessions("auth-session", store))

	// Routes
	routes.RegisterRoutes(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Serveur démarré sur :%s\n", port)
	http.ListenAndServe(":"+port, r)
}