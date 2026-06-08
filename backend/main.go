package main

import (
	"log"
	"net/http"
	"os"

	"violin-quest-api/db"
	"violin-quest-api/handlers"
	"violin-quest-api/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	db.Open()
	db.Migrate()
	db.Seed()

	r := gin.Default()
	r.Use(corsMiddleware())

	api := r.Group("/api")
	{
		api.GET("/options", handlers.GetOptions)
		api.GET("/stats", handlers.GetStats)
		api.POST("/session", handlers.SubmitSession)

		api.POST("/auth/login", handlers.Login)
		api.POST("/auth/refresh", handlers.RefreshToken)

		admin := api.Group("/admin")
		admin.Use(middleware.JWTAuth())
		{
			admin.GET("/sessions", handlers.ListPendingSessions)
			admin.GET("/history", handlers.GetHistory)
			admin.POST("/approve/:id", handlers.ApproveSession)
			admin.POST("/reject/:id", handlers.RejectSession)

			admin.GET("/children", handlers.ListChildren)
			admin.PATCH("/children/:id", handlers.RenameChild)

			admin.GET("/admins", handlers.ListAdmins)
			admin.POST("/admins", handlers.CreateAdmin)
			admin.DELETE("/admins/:id", handlers.DeleteAdmin)
			admin.PATCH("/admins/:id", handlers.UpdateAdmin)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("server: listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE, PATCH, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
