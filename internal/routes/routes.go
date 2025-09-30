package routes

import (
	"cedra_back_end/internal/handlers"
	"cedra_back_end/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	// Groupement des routes d’auth sous /api/auth
	auth := r.Group("/api/auth")
	{
		// Auth classique
		auth.POST("/register", handlers.CreateUser)
		auth.POST("/login", handlers.Login)

		// OAuth (Google / Facebook / Apple)
		auth.GET("/:provider", handlers.BeginAuth)
		auth.GET("/:provider/callback", handlers.CallbackAuth)

		// ✅ Routes protégées par JWT
		protected := auth.Group("/")
		protected.Use(middleware.AuthRequired())
		protected.GET("/me", handlers.Me)
	}
}
