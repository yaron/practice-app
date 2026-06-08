package handlers

import (
	"net/http"
	"strings"

	vqauth "violin-quest-api/auth"
	"violin-quest-api/db"

	"github.com/gin-gonic/gin"
)

type registerFCMRequest struct {
	Token   string `json:"token" binding:"required,max=4096"`
	Role    string `json:"role" binding:"required"`
	ChildID *int64 `json:"child_id"`
}

// RegisterFCMToken upserts an FCM registration token.
// CHILD registrations are unauthenticated but validate that the child_id exists.
// ADMIN registrations require a valid JWT in the Authorization header.
// POST /api/fcm/register
func RegisterFCMToken(c *gin.Context) {
	var req registerFCMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Role != "CHILD" && req.Role != "ADMIN" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be CHILD or ADMIN"})
		return
	}

	if req.Role == "ADMIN" {
		// ADMIN registrations must be authenticated.
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token ontbreekt"})
			return
		}
		if _, err := vqauth.ParseAdminID(strings.TrimPrefix(header, "Bearer ")); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "ongeldig token"})
			return
		}
	}

	if req.Role == "CHILD" {
		if req.ChildID == nil || *req.ChildID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "child_id vereist voor CHILD rol"})
			return
		}
		// Verify the child exists.
		var exists int
		if err := db.DB.QueryRow(`SELECT 1 FROM children WHERE id = ?`, *req.ChildID).Scan(&exists); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "onbekend child_id"})
			return
		}
	}

	var childID *int64
	if req.Role == "CHILD" {
		childID = req.ChildID
	}

	_, err := db.DB.Exec(`
		INSERT INTO fcm_tokens (token, child_id, role, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(token) DO UPDATE SET
			child_id   = excluded.child_id,
			role       = excluded.role,
			updated_at = CURRENT_TIMESTAMP
	`, req.Token, childID, req.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"registered": true})
}
