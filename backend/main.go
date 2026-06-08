package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"violin-quest-api/db"
	"violin-quest-api/handlers"
	"violin-quest-api/middleware"
	"violin-quest-api/reminder"

	"github.com/gin-gonic/gin"
)

func main() {
	if os.Getenv("JWT_SECRET") == "" {
		log.Printf("WARNING: JWT_SECRET is not set — using insecure development secret. Set JWT_SECRET in production.")
	}

	db.Open()
	db.Migrate()
	db.Seed()
	reminder.StartReminders()

	r := gin.Default()
	r.Use(corsMiddleware())

	api := r.Group("/api")
	{
		api.GET("/options", handlers.GetOptions)
		api.GET("/stats", handlers.GetStats)
		api.POST("/session", handlers.SubmitSession)

		api.POST("/auth/login", handlers.Login)
		api.POST("/auth/refresh", handlers.RefreshToken)
		api.POST("/fcm/register", handlers.RegisterFCMToken)

		admin := api.Group("/admin")
		admin.Use(middleware.JWTAuth())
		{
			admin.GET("/sessions", handlers.ListPendingSessions)
			admin.GET("/history", handlers.GetHistory)
			admin.POST("/approve/:id", handlers.ApproveSession)
			admin.POST("/reject/:id", handlers.RejectSession)
			admin.DELETE("/sessions/:id", handlers.DeleteSession)

			admin.GET("/children", handlers.ListChildren)
			admin.POST("/children", handlers.CreateChild)
			admin.PATCH("/children/:id", handlers.RenameChild)
			admin.DELETE("/children/:id", handlers.DeleteChild)

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
	// ALLOWED_ORIGINS: comma-separated list of allowed origins.
	// In release mode, defaults to the production frontend if not explicitly set.
	// In any other mode, defaults to "*".
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" && gin.Mode() == gin.ReleaseMode {
		allowedOrigins = "https://viool.naomital.nl"
	}
	originSet := map[string]bool{}
	useWildcard := allowedOrigins == ""
	if !useWildcard {
		for _, o := range strings.Split(allowedOrigins, ",") {
			if o = strings.TrimSpace(o); o != "" {
				originSet[o] = true
			}
		}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if useWildcard {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if originSet[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE, PATCH, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
