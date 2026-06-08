package main

import (
	"log"
	"net/http"
	"os"

	"violin-quest-api/db"
	"violin-quest-api/handlers"

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

// corsMiddleware allows requests from any origin during development.
// In production, Nginx handles CORS so this never fires.
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
