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
	addresses.Use(middleware.AuthRequired())
	{
		addresses.GET("/mine", handlers.ListMyAddresses)
		addresses.POST("", handlers.CreateAddress)
		addresses.DELETE("/:id", handlers.DeleteAddress)
		addresses.POST("/:id/default", handlers.MakeDefaultAddress)
	}

	// --- COMPANY ---
	company := r.Group("/api/company")
	company.Use(middleware.AuthRequired())
	{
		// Infos de la société
		company.GET("/me", handlers.GetMyCompany)

		// Gestion de la société (admin uniquement)
		admin := company.Group("/")
		admin.Use(middleware.CompanyAdminRequired())
		{
			// Adresse de facturation
			admin.PUT("/billing", handlers.UpdateCompanyBilling)
			
			// Gestion des employés
			admin.GET("/employees", handlers.ListCompanyEmployees)
			admin.POST("/employees", handlers.AddCompanyEmployee)
			admin.DELETE("/employees/:userId", handlers.RemoveCompanyEmployee)
			admin.PUT("/employees/:userId/admin", handlers.ToggleEmployeeAdmin)
		}
	}
}