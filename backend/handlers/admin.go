package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"violin-quest-api/db"
	"violin-quest-api/models"

	"github.com/gin-gonic/gin"
)

// ListPendingSessions returns all PENDING sessions with child names joined.
// GET /api/admin/sessions
func ListPendingSessions(c *gin.Context) {
	rows, err := db.DB.Query(
		`SELECT s.id, s.child_id, c.name, s.date, s.tasks_completed, s.status, s.created_at
		 FROM sessions s
		 JOIN children c ON c.id = s.child_id
		 WHERE s.status = 'PENDING'
		 ORDER BY s.created_at DESC`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer rows.Close()

	var sessions []models.PendingSession
	for rows.Next() {
		var ps models.PendingSession
		if err := rows.Scan(
			&ps.ID, &ps.ChildID, &ps.ChildName, &ps.Date,
			&ps.TasksCompleted, &ps.Status, &ps.CreatedAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		sessions = append(sessions, ps)
	}
	if sessions == nil {
		sessions = []models.PendingSession{}
	}
	c.JSON(http.StatusOK, sessions)
}

// ApproveSession runs the full approval workflow inside a SQLite transaction.
// POST /api/admin/approve/:id
func ApproveSession(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ongeldig id"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer tx.Rollback() //nolint:errcheck

	// 1. Fetch the session
	var s models.Session
	if err := tx.QueryRow(
		`SELECT id, child_id, date, tasks_completed, status FROM sessions WHERE id = ?`, id,
	).Scan(&s.ID, &s.ChildID, &s.Date, &s.TasksCompleted, &s.Status); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "sessie niet gevonden"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		}
		return
	}
	if s.Status != "PENDING" {
		c.JSON(http.StatusConflict, gin.H{"error": "sessie is niet meer in behandeling"})
		return
	}

	// 2. Is this the first approved session for this child today?
	var approvedToday int
	tx.QueryRow( //nolint:errcheck
		`SELECT COUNT(*) FROM sessions WHERE child_id = ? AND date = ? AND status = 'APPROVED'`,
		s.ChildID, s.Date,
	).Scan(&approvedToday)
	isFirstOfDay := approvedToday == 0

	firstOfDayInt := 0
	if isFirstOfDay {
		firstOfDayInt = 1
	}

	// 3+4. Mark approved
	if _, err := tx.Exec(
		`UPDATE sessions SET status = 'APPROVED', is_first_of_day = ? WHERE id = ?`,
		firstOfDayInt, id,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// 5. Base points
	basePoints := s.TasksCompleted * 10
	bonusPoints := 0

	// 6. Streak logic — only when this is the first approval of the day
	if isFirstOfDay {
		sessionDate, parseErr := time.Parse("2006-01-02", s.Date)
		if parseErr != nil {
			sessionDate = time.Now()
		}
		year, week := sessionDate.ISOWeek()
		yearWeek := isoWeekKey(year, week)

		if _, err := tx.Exec(
			`INSERT INTO weekly_streaks (child_id, year_week, session_count)
			 VALUES (?, ?, 1)
			 ON CONFLICT(child_id, year_week) DO UPDATE SET session_count = session_count + 1`,
			s.ChildID, yearWeek,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}

		var sessionCount, milestoneReached int
		tx.QueryRow( //nolint:errcheck
			`SELECT session_count, milestone_reached FROM weekly_streaks
			 WHERE child_id = ? AND year_week = ?`,
			s.ChildID, yearWeek,
		).Scan(&sessionCount, &milestoneReached)

		if sessionCount >= 3 && milestoneReached == 0 {
			bonusPoints = 50
			if _, err := tx.Exec(
				`UPDATE weekly_streaks SET milestone_reached = 1
				 WHERE child_id = ? AND year_week = ?`,
				s.ChildID, yearWeek,
			); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
				return
			}
		}
	}

	totalNew := basePoints + bonusPoints

	// 7. Load and update user_stats
	var stats models.UserStats
	if err := tx.QueryRow(
		`SELECT id, total_points, current_level, experience_points FROM user_stats WHERE child_id = ?`,
		s.ChildID,
	).Scan(&stats.ID, &stats.TotalPoints, &stats.CurrentLevel, &stats.ExperiencePoints); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	stats.TotalPoints += totalNew
	stats.ExperiencePoints += totalNew

	// Level-up loop
	threshold := stats.CurrentLevel * 100
	for stats.ExperiencePoints >= threshold {
		stats.ExperiencePoints -= threshold
		stats.CurrentLevel++
		threshold = stats.CurrentLevel * 100
	}

	if _, err := tx.Exec(
		`UPDATE user_stats SET total_points = ?, current_level = ?, experience_points = ? WHERE id = ?`,
		stats.TotalPoints, stats.CurrentLevel, stats.ExperiencePoints, stats.ID,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"approved":     true,
		"base_points":  basePoints,
		"bonus_points": bonusPoints,
	})
}

// GetHistory returns all sessions (any status) with their individual task names.
// GET /api/admin/history
func GetHistory(c *gin.Context) {
	// One query: sessions joined with children + task names aggregated per session.
	// GROUP_CONCAT uses char(31) (ASCII Unit Separator) as delimiter — safe because
	// task texts are user-readable strings that will never contain this character.
	rows, err := db.DB.Query(`
		SELECT s.id, s.child_id, c.name, s.date, s.tasks_completed, s.status,
		       COALESCE(s.rejection_note, ''), s.created_at,
		       GROUP_CONCAT(st.task_text, char(31))
		FROM sessions s
		JOIN children c ON c.id = s.child_id
		LEFT JOIN session_tasks st ON st.session_id = s.id
		GROUP BY s.id
		ORDER BY s.created_at DESC`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer rows.Close()

	var sessions []models.SessionWithTasks
	for rows.Next() {
		var s models.SessionWithTasks
		var tasksConcat sql.NullString
		if err := rows.Scan(
			&s.ID, &s.ChildID, &s.ChildName, &s.Date, &s.TasksCompleted,
			&s.Status, &s.RejectionNote, &s.CreatedAt, &tasksConcat,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		if tasksConcat.Valid && tasksConcat.String != "" {
			s.Tasks = strings.Split(tasksConcat.String, "\x1f")
		} else {
			s.Tasks = []string{}
		}
		sessions = append(sessions, s)
	}
	if sessions == nil {
		sessions = []models.SessionWithTasks{}
	}
	c.JSON(http.StatusOK, sessions)
}

type rejectRequest struct {
	Note string `json:"note"`
}

// RejectSession marks a PENDING session as REJECTED with an optional parent note.
// POST /api/admin/reject/:id
func RejectSession(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ongeldig id"})
		return
	}

	var req rejectRequest
	c.ShouldBindJSON(&req) //nolint:errcheck — note is optional

	res, err := db.DB.Exec(
		`UPDATE sessions SET status = 'REJECTED', rejection_note = ?
		 WHERE id = ? AND status = 'PENDING'`,
		req.Note, id,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "sessie niet gevonden of niet in behandeling"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"rejected": true})
}
