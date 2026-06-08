package handlers

import (
	"net/http"
	"time"

	"violin-quest-api/db"

	"github.com/gin-gonic/gin"
)

type submitSessionRequest struct {
	ChildID        int64 `json:"child_id" binding:"required"`
	TasksCompleted int   `json:"tasks_completed" binding:"required,min=1"`
}

// SubmitSession creates a new PENDING session for a child.
// POST /api/session
func SubmitSession(c *gin.Context) {
	var req submitSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify the child exists
	var exists int
	if err := db.DB.QueryRow(
		`SELECT COUNT(*) FROM children WHERE id = ?`, req.ChildID,
	).Scan(&exists); err != nil || exists == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown child_id"})
		return
	}

	today := time.Now().Format("2006-01-02")

	res, err := db.DB.Exec(
		`INSERT INTO sessions (child_id, date, tasks_completed, status) VALUES (?, ?, ?, 'PENDING')`,
		req.ChildID, today, req.TasksCompleted,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	id, _ := res.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"id": id})
}
