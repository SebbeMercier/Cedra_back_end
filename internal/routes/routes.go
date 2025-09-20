package routes

import (
	"github.com/gin-gonic/gin"
	"cedra_back_end/internal/handlers"
)

func RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "pong"})
		})

		// Exemples de routes internes (handlers maison)
		api.GET("/users", handlers.GetUsers)
		api.POST("/users", handlers.CreateUser)
	}
}
