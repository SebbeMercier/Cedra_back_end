package routes

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"cedra_back_end/internal/handlers/company"
	"cedra_back_end/internal/handlers/payement"
	"cedra_back_end/internal/handlers/product"
	"cedra_back_end/internal/handlers/user"
	"cedra_back_end/internal/middleware"
)

func RegisterRoutes(router *gin.Engine) {
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:5173"}, // ✅ Spécifier les origines exactes
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	api := router.Group("/api")

	auth := api.Group("/auth")
	{
		// 🔹 Auth locale
		auth.POST("/register", user.CreateUser)
		auth.POST("/login", user.Login)
		auth.GET("/me", middleware.AuthRequired(), user.Me)

		auth.POST("/merge", middleware.AuthRequired(), user.MergeAccount)
		auth.POST("/complete", middleware.AuthRequired(), user.CompleteProfile)
		auth.POST("/change-password", middleware.AuthRequired(), user.ChangePassword)
		auth.DELETE("/delete-account", middleware.AuthRequired(), user.DeleteAccount)

		auth.POST("/forgot-password", user.ForgotPassword)
		auth.POST("/reset-password", user.ResetPassword)

		auth.POST("/google/mobile", user.GoogleMobileLogin)
		auth.POST("/facebook/mobile", user.FacebookMobileLogin)

		auth.GET("/:provider", user.BeginAuth)
		auth.GET("/:provider/callback", user.CallbackAuth)
	}

	addresses := api.Group("/addresses", middleware.AuthRequired())
	{
		addresses.GET("/mine", user.ListMyAddresses)
		addresses.POST("", user.CreateAddress)
		addresses.DELETE("/:id", user.DeleteAddress)
		addresses.POST("/:id/default", user.MakeDefaultAddress)
	}

	orders := api.Group("/orders", middleware.AuthRequired())
	{
		orders.GET("/mine", user.GetMyOrders)
		orders.GET("/:id", user.GetOrderByID)
	}

	companyGroup := api.Group("/company", middleware.AuthRequired())
	{
		companyGroup.GET("/me", company.GetMyCompany)
		companyGroup.PUT("/billing", middleware.CompanyAdminRequired(), company.UpdateCompanyBilling)
		companyGroup.GET("/employees", middleware.CompanyAdminRequired(), company.ListCompanyEmployees)
		companyGroup.POST("/employees", middleware.CompanyAdminRequired(), company.AddCompanyEmployee)
		companyGroup.DELETE("/employees/:userId", middleware.CompanyAdminRequired(), company.RemoveCompanyEmployee)
		companyGroup.PUT("/employees/:userId/admin", middleware.CompanyAdminRequired(), company.ToggleEmployeeAdmin)
	}

	// =========================
	// 🛍️ PRODUCTS
	// =========================
	products := api.Group("/products")
	{
		products.GET("", product.GetAllProducts)
		products.GET("/search", product.SearchProducts)
		products.GET("/category/:id", product.GetProductsByCategory)
		products.GET("/best-sellers", product.GetBestSellers)
		products.GET("/:id", product.GetProductFull)

		// 🔹 Routes admin (création)
		products.POST("", middleware.AuthRequired(), middleware.RequireAdmin, product.CreateProduct)
		products.PUT("/:id", middleware.AuthRequired(), middleware.RequireAdmin, product.UpdateProduct)
		products.DELETE("/:id", middleware.AuthRequired(), middleware.RequireAdmin, product.DeleteProduct)
	}

	categories := api.Group("/categories")
	{
		categories.GET("", product.GetAllCategories)
		categories.POST("", middleware.AuthRequired(), middleware.RequireAdmin, product.CreateCategory)
		categories.PUT("/:id", middleware.AuthRequired(), middleware.RequireAdmin, product.UpdateCategory)
		categories.DELETE("/:id", middleware.AuthRequired(), middleware.RequireAdmin, product.DeleteCategory)
	}

	cart := api.Group("/cart", middleware.AuthRequired())
	{
		cart.GET("", user.GetCart)
		cart.POST("/add", user.AddToCart)
		cart.PUT("/:productId", user.UpdateCartQuantity)
		cart.DELETE("/:productId", user.RemoveFromCart)
		cart.DELETE("", user.ClearCart)
	}

	images := api.Group("/images")
	{
		images.GET("/:productId", product.GetProductImages)
		images.POST("/upload", middleware.AuthRequired(), middleware.RequireAdmin, product.UploadProductImage)
		images.DELETE("/:id", middleware.AuthRequired(), middleware.RequireAdmin, product.DeleteProductImage)
	}

	payments := api.Group("/payments")
	{
		payments.POST("/create-intent", middleware.AuthRequired(), payement.CreatePaymentIntent)
		payments.POST("/webhook", payement.StripeWebhook) // ⚠️ Pas d'auth (Stripe vérifie la signature)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"version": "1.0.0",
		})
	})
}
