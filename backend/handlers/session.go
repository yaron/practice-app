package handlers

import (
	"fmt"
	"net/http"
	"time"

	"violin-quest-api/db"
	"violin-quest-api/fcm"

	"github.com/gin-gonic/gin"
)

// childLocalDate returns today's date string (YYYY-MM-DD) in the child's IANA timezone.
// Falls back to UTC if the timezone is missing or unrecognised.
func childLocalDate(childID int64) string {
	var tz string
	db.DB.QueryRow(`SELECT timezone FROM children WHERE id = ?`, childID).Scan(&tz) //nolint:errcheck
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	return time.Now().In(loc).Format("2006-01-02")
}

type submitSessionRequest struct {
	ChildID int64 `json:"child_id" binding:"required"`
	// Tasks is the preferred field: individual task names.
	// tasks_completed is accepted as a fallback for backward compatibility.
	Tasks          []string `json:"tasks"`
	TasksCompleted int      `json:"tasks_completed"`
}

// SubmitSession creates a new PENDING session for a child.
// POST /api/session
func SubmitSession(c *gin.Context) {
	var req submitSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	count := req.TasksCompleted
	if len(req.Tasks) > 0 {
		count = len(req.Tasks)
	}
	if count < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "minimaal één taak vereist"})
		return
	}

	var childName string
	if err := db.DB.QueryRow(
		`SELECT name FROM children WHERE id = ?`, req.ChildID,
	).Scan(&childName); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown child_id"})
		return
	}

	today := childLocalDate(req.ChildID)

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer tx.Rollback() //nolint:errcheck

	res, err := tx.Exec(
		`INSERT INTO sessions (child_id, date, tasks_completed, status) VALUES (?, ?, ?, 'PENDING')`,
		req.ChildID, today, count,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	sessionID, _ := res.LastInsertId()

	for _, task := range req.Tasks {
		if len(task) > 200 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "taaknaam te lang (max 200 tekens)"})
			return
		}
		if _, err := tx.Exec(
			`INSERT INTO session_tasks (session_id, task_text) VALUES (?, ?)`,
			sessionID, task,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	name := childName
	taskCount := count
	if len(req.Tasks) > 0 {
		taskCount = len(req.Tasks)
	}
	go func() {
		tokens := db.GetAdminTokens()
		fcm.Send(tokens, "🎻 "+name+" heeft geoefend!", fmt.Sprintf("%d taken afgerond", taskCount))
	}()
	// Note: fcm.Send is a no-op when FCM_PROJECT_ID is unset.

	c.JSON(http.StatusCreated, gin.H{"id": sessionID})
}
