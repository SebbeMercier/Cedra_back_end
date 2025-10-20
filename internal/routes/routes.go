package routes

import (
	"cedra_back_end/internal/handlers"
	"cedra_back_end/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	// --- Authentification ---
	auth := r.Group("/api/auth")
	{
		auth.POST("/register", handlers.CreateUser)
		auth.POST("/login", handlers.Login)

		auth.GET("/:provider", handlers.BeginAuth)
		auth.GET("/:provider/callback", handlers.CallbackAuth)

		protected := auth.Group("/")
		protected.Use(middleware.AuthRequired())
		protected.GET("/me", handlers.Me)
	}

	// --- Adresses ---
	addresses := r.Group("/api/addresses")
	addresses.Use(middleware.AuthRequired())
	{
		addresses.GET("/mine", handlers.ListMyAddresses)
		addresses.POST("", handlers.CreateAddress)
		addresses.DELETE("/:id", handlers.DeleteAddress)
		addresses.POST("/:id/default", handlers.MakeDefaultAddress)
	}

	// --- Entreprise ---
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

	RegisterProductRoutes(r)
	RegisterCategoryRoutes(r)
	RegisterCartRoutes(r)
	RegisterImageRoutes(r)

	// 🆕 ✅ N’oublie pas cette ligne :
	RegisterPaymentRoutes(r)
}
// PRODUITS
func RegisterProductRoutes(r *gin.Engine) {
	api := r.Group("/api/products")
	{
		api.GET("/", handlers.GetAllProducts)
		api.GET("/search", handlers.SearchProducts)
		api.GET("/category/:id", handlers.GetProductsByCategory) // facultatif
		api.POST("/", middleware.AuthRequired(), middleware.RequireAdmin, handlers.CreateProduct)
		api.GET("/:id/full", handlers.GetProductFull)
	}
}

// CATEGORIES
func RegisterCategoryRoutes(r *gin.Engine) {
	api := r.Group("/api/categories")
	{
		api.GET("/", handlers.GetAllCategories)
		api.POST("/", middleware.AuthRequired(), handlers.CreateCategory)
	}
}

// PANIER
func RegisterCartRoutes(r *gin.Engine) {
	cart := r.Group("/api/cart")
	cart.Use(middleware.AuthRequired())
	{
		cart.GET("/", handlers.GetCart)
		cart.POST("/add", handlers.AddToCart)
		cart.DELETE("/:productId", handlers.RemoveFromCart)
		cart.DELETE("/clear", handlers.ClearCart) // 🧹 Nouveau endpoint pour vider le panier
	}
}

// IMAGES
func RegisterImageRoutes(r *gin.Engine) {
	images := r.Group("/api/images")

	// 🔓 Lecture publique
	images.GET("/:productId", handlers.GetProductImages)

	// 🔐 Actions protégées (upload + suppression)
	protected := images.Group("/")
	protected.Use(middleware.AuthRequired())
	{
		protected.POST("/upload", handlers.UploadProductImage)
		protected.DELETE("/:id", handlers.DeleteProductImage)
	}
}

func RegisterPaymentRoutes(r *gin.Engine) {
	payment := r.Group("/api/payments")
	payment.Use(middleware.AuthRequired())
	{
		payment.POST("/create-intent", handlers.CreatePaymentIntent)
	}

	// Webhook Stripe (public)
	r.POST("/api/payments/webhook", handlers.StripeWebhook)
}