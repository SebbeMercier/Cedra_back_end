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
		company.GET("/me", handlers.GetMyCompany)

		admin := company.Group("/")
		admin.Use(middleware.CompanyAdminRequired())
		{
			admin.PUT("/billing", handlers.UpdateCompanyBilling)
			admin.GET("/employees", handlers.ListCompanyEmployees)
			admin.POST("/employees", handlers.AddCompanyEmployee)
			admin.DELETE("/employees/:userId", handlers.RemoveCompanyEmployee)
			admin.PUT("/employees/:userId/admin", handlers.ToggleEmployeeAdmin)
		}
	}

	// --- PRODUCTS ---
	RegisterProductRoutes(r)

	// --- CATEGORIES ---
	RegisterCategoryRoutes(r)
}

// ✅ Module PRODUITS
func RegisterProductRoutes(r *gin.Engine) {
	api := r.Group("/api/products")
	{
		api.GET("/", handlers.GetAllProducts)
		api.GET("/search", handlers.SearchProducts)
		api.POST("/", middleware.RequireAdmin, handlers.CreateProduct)
	}

	cart := r.Group("/api/cart")
	cart.Use(middleware.AuthJWT)
	{
		cart.POST("/add", handlers.AddToCart)
		cart.GET("/", handlers.GetCart)
	}
}

// ✅ Module CATÉGORIES
func RegisterCategoryRoutes(r *gin.Engine) {
	api := r.Group("/api/categories")
	{
		api.GET("/", handlers.GetAllCategories)
		api.POST("/", middleware.AuthRequired(), handlers.CreateCategory)
	}
}
