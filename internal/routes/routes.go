package routes

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"cedra_back_end/internal/handlers/company"
	pa "cedra_back_end/internal/handlers/payement"
	"cedra_back_end/internal/handlers/product"
	"cedra_back_end/internal/handlers/user"
	"cedra_back_end/internal/middleware"
)

func RegisterRoutes(router *gin.Engine) {
	router.Use(cors.New(cors.Config{
		AllowAllOrigins:  true, // ✅ Permet toutes les origines (dev uniquement)
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "Content-Encoding"},
		AllowCredentials: false,          // ⚠️ Doit être false avec AllowAllOrigins
		MaxAge:           24 * time.Hour, // ✅ Augmenté à 24h pour réduire les preflight
	}))

	api := router.Group("/api")

	// ✅ Rate limiting global sur toutes les routes API
	api.Use(middleware.APIRateLimit())

	auth := api.Group("/auth")
	{
		// 🔹 Auth locale avec rate limiting
		auth.POST("/register", middleware.RegisterRateLimit(), user.CreateUser)
		auth.POST("/login", middleware.LoginRateLimit(), user.Login)
		auth.GET("/me", middleware.AuthRequired(), user.Me)

		auth.POST("/merge", middleware.AuthRequired(), user.MergeAccount)
		auth.POST("/complete", middleware.AuthRequired(), user.CompleteProfile)
		auth.POST("/change-password", middleware.AuthRequired(), user.ChangePassword)
		auth.DELETE("/delete-account", middleware.AuthRequired(), user.DeleteAccount)

		auth.POST("/forgot-password", middleware.ForgotPasswordRateLimit(), user.ForgotPassword)
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
		products.GET("/search", middleware.SearchRateLimit(), product.SearchProducts)
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
		cart.GET("", user.GetCartOptimized)                                    // ✅ Optimisé
		cart.GET("/sync", user.SyncCart)                                       // ✅ Nouveau - Synchronisation
		cart.POST("/add", middleware.CartRateLimit(), user.AddToCartOptimized) // ✅ Optimisé
		cart.PUT("/:productId", user.UpdateCartQuantityOptimized)              // ✅ Optimisé
		cart.DELETE("/:productId", user.RemoveFromCartOptimized)               // ✅ Optimisé
		cart.DELETE("", user.ClearCartOptimized)                               // ✅ Optimisé
	}

	images := api.Group("/images")
	{
		images.GET("/:productId", product.GetProductImages)
		images.POST("/upload", middleware.AuthRequired(), middleware.RequireAdmin, product.UploadProductImage)
		images.DELETE("/:id", middleware.AuthRequired(), middleware.RequireAdmin, product.DeleteProductImage)
	}

	payments := api.Group("/payments")
	{
		payments.POST("/create-intent", middleware.AuthRequired(), pa.CreatePaymentIntent)
		payments.POST("/checkout", middleware.AuthRequired(), pa.Checkout)             // ✅ Nouveau endpoint checkout
		payments.GET("/validate-coupon", middleware.AuthRequired(), pa.ValidateCoupon) // ✅ Validation coupon
		payments.POST("/webhook", pa.StripeWebhook)                                    // ⚠️ Pas d'auth (Stripe vérifie la signature)
	}

	// ✅ Routes admin pour la gestion des commandes
	adminOrders := api.Group("/admin/orders", middleware.AuthRequired(), middleware.RequireAdmin)
	{
		adminOrders.GET("", pa.GetAllOrders)
		adminOrders.GET("/stats", pa.GetOrderStats)
		adminOrders.PUT("/:id/status", pa.UpdateOrderStatus)
	}

	// ✅ Wishlist
	wishlist := api.Group("/wishlist", middleware.AuthRequired())
	{
		wishlist.GET("", user.GetWishlist)
		wishlist.POST("/add", user.AddToWishlist)
		wishlist.DELETE("/:productId", user.RemoveFromWishlist)
	}

	// ✅ Reviews & Ratings
	reviews := api.Group("/reviews")
	{
		reviews.GET("/product/:id", product.GetProductReviews)
		reviews.POST("/product/:id", middleware.AuthRequired(), product.CreateReview)
	}

	// ✅ Refunds
	refunds := api.Group("/refunds", middleware.AuthRequired())
	{
		refunds.GET("/mine", pa.GetUserRefunds)
		refunds.POST("/order/:orderId", pa.RequestRefund)
	}

	adminRefunds := api.Group("/admin/refunds", middleware.AuthRequired(), middleware.RequireAdmin)
	{
		adminRefunds.GET("", pa.GetAllRefunds)
		adminRefunds.PUT("/:refundId/process", pa.ProcessRefund)
	}

	// ✅ Shipping
	shipping := api.Group("/shipping")
	{
		shipping.GET("/options", pa.GetShippingOptions)
	}

	// ✅ Advanced Search
	search := api.Group("/search")
	{
		search.GET("/advanced", product.SearchProductsAdvanced)
		search.GET("/filters", product.GetProductFilters)
	}

	// ✅ Dashboard Admin
	dashboard := api.Group("/admin/dashboard", middleware.AuthRequired(), middleware.RequireAdmin)
	{
		dashboard.GET("/stats", pa.GetDashboardStats)
		dashboard.GET("/recent-orders", pa.GetRecentOrders)
		dashboard.GET("/top-products", pa.GetTopProducts)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"version": "1.1.0",
		})
	})
}
