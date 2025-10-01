package routes

import (
	"cedra_back_end/internal/handlers"
	"cedra_back_end/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	// --- AUTH ---
	auth := r.Group("/api/auth")
	{
		auth.POST("/register", handlers.CreateUser)
		auth.POST("/login", handlers.Login)

		// OAuth
		auth.GET("/:provider", handlers.BeginAuth)
		auth.GET("/:provider/callback", handlers.CallbackAuth)

		protected := auth.Group("/")
		protected.Use(middleware.AuthRequired())
		protected.GET("/me", handlers.Me)
	}

	// --- ADDRESSES ---
	addresses := r.Group("/api/addresses")
	addresses.Use(middleware.AuthRequired()) // ⚡ Protégé par JWT
	{
		addresses.GET("/mine", handlers.ListMyAddresses)   // GET mes adresses
		addresses.POST("", handlers.CreateAddress)         // POST nouvelle adresse
		addresses.DELETE("/:id", handlers.DeleteAddress)   // DELETE une adresse
		addresses.POST("/:id/default", handlers.MakeDefaultAddress) // POST définir par défaut
	}
}
