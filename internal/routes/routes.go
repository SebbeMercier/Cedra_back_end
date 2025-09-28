package routes

import (
	"cedra_back_end/internal/handlers"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	// Users
	r.GET("/users", handlers.GetUsers)
	r.POST("/users", handlers.CreateUser)
}
