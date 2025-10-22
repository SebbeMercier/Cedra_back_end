package routes

import (
	"cedra_back_end/internal/handlers/product"
	"cedra_back_end/internal/handlers/user"
	"cedra_back_end/internal/handlers/company"
	"cedra_back_end/internal/handlers/payement"
	"cedra_back_end/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	// --- Authentification ---
	auth := r.Group("/api/auth")
	{
		auth.POST("/register", user.CreateUser)
		auth.POST("/login", user.Login)

		auth.GET("/:provider", user.BeginAuth)
		auth.GET("/:provider/callback", user.CallbackAuth)

		protected := auth.Group("/")
		protected.Use(middleware.AuthRequired())
		protected.GET("/me", user.Me)
	}

	// --- Adresses ---
	addresses := r.Group("/api/addresses")
	addresses.Use(middleware.AuthRequired())
	{
		addresses.GET("/mine", user.ListMyAddresses)
		addresses.POST("", user.CreateAddress)
		addresses.DELETE("/:id", user.DeleteAddress)
		addresses.POST("/:id/default", user.MakeDefaultAddress)
	}

	// --- Entreprise ---
	company := r.Group("/api/company")
	company.Use(middleware.AuthRequired())
	{
		company.GET("/me", Company.GetMyCompany)

		admin := company.Group("/")
		admin.Use(middleware.CompanyAdminRequired())
		{
			admin.PUT("/billing", Company.UpdateCompanyBilling)
			admin.GET("/employees", Company.ListCompanyEmployees)
			admin.POST("/employees", Company.AddCompanyEmployee)
			admin.DELETE("/employees/:userId", Company.RemoveCompanyEmployee)
			admin.PUT("/employees/:userId/admin", Company.ToggleEmployeeAdmin)
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
		api.GET("/", product.GetAllProducts)
		api.GET("/search", product.SearchProducts)
		api.GET("/category/:id", product.GetProductsByCategory) // facultatif
		api.POST("/", middleware.AuthRequired(), middleware.RequireAdmin, product.CreateProduct)
		api.GET("/:id/full", product.GetProductFull)
	}
}

// CATEGORIES
func RegisterCategoryRoutes(r *gin.Engine) {
	api := r.Group("/api/categories")
	{
		api.GET("/", product.GetAllCategories)
		api.POST("/", middleware.AuthRequired(), product.CreateCategory)
	}
}

// PANIER
func RegisterCartRoutes(r *gin.Engine) {
	cart := r.Group("/api/cart")
	cart.Use(middleware.AuthRequired())
	{
		cart.GET("/", user.GetCart)
		cart.POST("/add", user.AddToCart)
		cart.DELETE("/:productId", user.RemoveFromCart)
		cart.DELETE("/clear", user.ClearCart) // 🧹 Nouveau endpoint pour vider le panier
	}
}

// IMAGES
func RegisterImageRoutes(r *gin.Engine) {
	images := r.Group("/api/images")

	// 🔓 Lecture publique
	images.GET("/:productId", product.GetProductImages)

	// 🔐 Actions protégées (upload + suppression)
	protected := images.Group("/")
	protected.Use(middleware.AuthRequired())
	{
		protected.POST("/upload", product.UploadProductImage)
		protected.DELETE("/:id", product.DeleteProductImage)
	}
}

func RegisterPaymentRoutes(r *gin.Engine) {
	payment := r.Group("/api/payments")
	payment.Use(middleware.AuthRequired())
	{
		payment.POST("/create-intent", payement.CreatePaymentIntent)
	}

	// Webhook Stripe (public)
	r.POST("/api/payments/webhook", payement.StripeWebhook)
}