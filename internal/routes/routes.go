package routes

import (
	"cedra_back_end/internal/handlers/Company"
	"cedra_back_end/internal/handlers/payement"
	"cedra_back_end/internal/handlers/product"
	"cedra_back_end/internal/handlers/user"
	"cedra_back_end/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api")

	// ========== AUTH ==========
	auth := api.Group("/auth")
	{
		auth.POST("/register", user.CreateUser)
		auth.POST("/login", user.Login)
		auth.GET("/:provider", user.BeginAuth)
		auth.GET("/:provider/callback", user.CallbackAuth)
		auth.GET("/me", middleware.AuthRequired(), user.Me)
	}

	// ========== ADDRESSES ==========
	addresses := api.Group("/addresses", middleware.AuthRequired())
	{
		addresses.GET("/mine", user.ListMyAddresses)
		addresses.POST("", user.CreateAddress)
		addresses.DELETE("/:id", user.DeleteAddress)
		addresses.POST("/:id/default", user.MakeDefaultAddress)
	}

	// ========== ORDERS ========== ✅ NOUVEAU
	orders := api.Group("/orders", middleware.AuthRequired())
	{
		orders.GET("/mine", user.GetMyOrders)        // Liste de toutes les commandes
		orders.GET("/:id", user.GetOrderByID)        // Détail d'une commande
	}

	// ========== COMPANY ==========
	companyGroup := api.Group("/company", middleware.AuthRequired())
	{
		companyGroup.GET("/me", Company.GetMyCompany)
		companyGroup.PUT("/billing", middleware.CompanyAdminRequired(), Company.UpdateCompanyBilling)
		companyGroup.GET("/employees", middleware.CompanyAdminRequired(), Company.ListCompanyEmployees)
		companyGroup.POST("/employees", middleware.CompanyAdminRequired(), Company.AddCompanyEmployee)
		companyGroup.DELETE("/employees/:userId", middleware.CompanyAdminRequired(), Company.RemoveCompanyEmployee)
		companyGroup.PUT("/employees/:userId/admin", middleware.CompanyAdminRequired(), Company.ToggleEmployeeAdmin)
	}

	// ========== PRODUCTS ==========
	products := api.Group("/products")
	{
		products.GET("/", product.GetAllProducts)
		products.GET("/search", product.SearchProducts)
		products.GET("/category/:id", product.GetProductsByCategory)
		products.POST("/", middleware.AuthRequired(), middleware.CompanyAdminRequired(), product.CreateProduct)
		products.GET("/:id/full", product.GetProductFull)
	}

	// ========== CATEGORIES ==========
	categories := api.Group("/categories")
	{
		categories.GET("/", product.GetAllCategories)
		categories.POST("/", middleware.AuthRequired(), product.CreateCategory)
	}

	// ========== CART ==========
	cart := api.Group("/cart", middleware.AuthRequired())
	{
		cart.GET("/", user.GetCart)
		cart.POST("/add", user.AddToCart)
		cart.DELETE("/:productId", user.RemoveFromCart)
		cart.DELETE("/clear", user.ClearCart)
	}

	// ========== IMAGES ==========
	images := api.Group("/images")
	{
		images.GET("/:productId", product.GetProductImages)
		images.POST("/upload", middleware.AuthRequired(), product.UploadProductImage)
		images.DELETE("/:id", middleware.AuthRequired(), product.DeleteProductImage)
	}

	// ========== PAYMENTS ==========
	payments := api.Group("/payments")
	{
		payments.POST("/create-intent", middleware.AuthRequired(), payement.CreatePaymentIntent)
		payments.POST("/webhook", payement.StripeWebhook)
	}
}