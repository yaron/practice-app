package middleware

import (
	"net/http"
	"strings"

	vqauth "violin-quest-api/auth"

	"github.com/gin-gonic/gin"
)

// JWTAuth validates the Authorization: Bearer <token> header.
// On success it sets "admin_id" (int64) in the Gin context.
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token ontbreekt"})
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		adminID, err := vqauth.ParseAdminID(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "ongeldig token"})
			return
		}
		c.Set("admin_id", adminID)
		c.Next()
	}
}
