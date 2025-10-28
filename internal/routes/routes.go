package routes

import (
	"cedra_back_end/internal/handlers/company"
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
		// 🔹 Auth locale
		auth.POST("/register", user.CreateUser)
		auth.POST("/login", user.Login)
		auth.GET("/me", middleware.AuthRequired(), user.Me)

		// 🔹 Routes protégées
		auth.POST("/merge", middleware.AuthRequired(), user.MergeAccount)
		auth.POST("/complete", middleware.AuthRequired(), user.CompleteProfile)
		auth.POST("/change-password", middleware.AuthRequired(), user.ChangePassword) // ✅ Avec ancien mot de passe

		// 🔹 Réinitialisation de mot de passe (SANS authentification)
		auth.POST("/forgot-password", user.ForgotPassword) // ✅ Demande de reset
		auth.POST("/reset-password", user.ResetPassword)   // ✅ Reset avec token

		// 🔹 Auth mobile (OAuth natif iOS / Android)
		auth.POST("/google/mobile", user.GoogleMobileLogin)
		auth.POST("/facebook/mobile", user.FacebookMobileLogin)

		// 🔹 Auth sociale (OAuth Web)
		auth.GET("/:provider", user.BeginAuth)
		auth.GET("/:provider/callback", user.CallbackAuth)
	}

	// ========== ADDRESSES ==========
	addresses := api.Group("/addresses", middleware.AuthRequired())
	{
		addresses.GET("/mine", user.ListMyAddresses)
		addresses.POST("", user.CreateAddress)
		addresses.DELETE("/:id", user.DeleteAddress)
		addresses.POST("/:id/default", user.MakeDefaultAddress)
	}

	// ========== ORDERS ==========
	orders := api.Group("/orders", middleware.AuthRequired())
	{
		orders.GET("/mine", user.GetMyOrders)
		orders.GET("/:id", user.GetOrderByID)
	}

	// ========== COMPANY ==========
	companyGroup := api.Group("/company", middleware.AuthRequired())
	{
		companyGroup.GET("/me", company.GetMyCompany)
		companyGroup.PUT("/billing", middleware.CompanyAdminRequired(), company.UpdateCompanyBilling)
		companyGroup.GET("/employees", middleware.CompanyAdminRequired(), company.ListCompanyEmployees)
		companyGroup.POST("/employees", middleware.CompanyAdminRequired(), company.AddCompanyEmployee)
		companyGroup.DELETE("/employees/:userId", middleware.CompanyAdminRequired(), company.RemoveCompanyEmployee)
		companyGroup.PUT("/employees/:userId/admin", middleware.CompanyAdminRequired(), company.ToggleEmployeeAdmin)
	}

	// ========== PRODUCTS ==========
	products := api.Group("/products")
	{
		products.GET("/", product.GetAllProducts)
		products.GET("/search", product.SearchProducts)
		products.GET("/category/:id", product.GetProductsByCategory)
		products.GET("/:id/full", product.GetProductFull)
		products.POST("/", middleware.AuthRequired(), middleware.CompanyAdminRequired(), product.CreateProduct)
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
