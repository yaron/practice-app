package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"violin-quest-api/db"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type childSummary struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	TotalPoints int    `json:"total_points"`
}

type childPublic struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// ListChildrenPublic returns id+name for all children (no auth required).
// GET /api/children
func ListChildrenPublic(c *gin.Context) {
	rows, err := db.DB.Query(`SELECT id, name FROM children ORDER BY id`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer rows.Close()

	var children []childPublic
	for rows.Next() {
		var ch childPublic
		if err := rows.Scan(&ch.ID, &ch.Name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		children = append(children, ch)
	}
	if children == nil {
		children = []childPublic{}
	}
	c.JSON(http.StatusOK, children)
}

// ListChildren returns all children with their total score.
// GET /api/admin/children
func ListChildren(c *gin.Context) {
	rows, err := db.DB.Query(
		`SELECT c.id, c.name, COALESCE(us.total_points, 0)
		 FROM children c
		 LEFT JOIN user_stats us ON us.child_id = c.id
		 ORDER BY c.id`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer rows.Close()

	var children []childSummary
	for rows.Next() {
		var ch childSummary
		if err := rows.Scan(&ch.ID, &ch.Name, &ch.TotalPoints); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		children = append(children, ch)
	}
	if children == nil {
		children = []childSummary{}
	}
	c.JSON(http.StatusOK, children)
}

type createChildRequest struct {
	Name     string `json:"name" binding:"required,max=100"`
	Timezone string `json:"timezone" binding:"max=100"`
}

// CreateChild creates a new child with user_stats and default wheel options.
// POST /api/admin/children
func CreateChild(c *gin.Context) {
	var req createChildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tz := req.Timezone
	if tz == "" {
		tz = "Europe/Amsterdam"
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer tx.Rollback() //nolint:errcheck

	res, err := tx.Exec(`INSERT INTO children (name, timezone) VALUES (?, ?)`, req.Name, tz)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	childID, _ := res.LastInsertId()

	if _, err := tx.Exec(`INSERT INTO user_stats (child_id) VALUES (?)`, childID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	for _, opt := range db.DefaultWheelOptions {
		if _, err := tx.Exec(
			`INSERT INTO wheel_options (child_id, text, short_text, is_bonus) VALUES (?, ?, ?, ?)`,
			childID, opt.Text, opt.ShortText, opt.IsBonus,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.JSON(http.StatusCreated, childSummary{ID: childID, Name: req.Name, TotalPoints: 0})
}

type renameChildRequest struct {
	Name string `json:"name" binding:"required,max=100"`
}

// RenameChild updates a child's display name.
// PATCH /api/admin/children/:id
func RenameChild(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ongeldig id"})
		return
	}

	var req renameChildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := db.DB.Exec(`UPDATE children SET name = ? WHERE id = ?`, req.Name, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "kind niet gevonden"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// DeleteChild removes a child and all their associated data (sessions, stats,
// wheel options, streaks, FCM tokens).
// DELETE /api/admin/children/:id
func DeleteChild(c *gin.Context) {
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

	var exists int
	if err := tx.QueryRow(`SELECT 1 FROM children WHERE id = ?`, id).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "kind niet gevonden"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		}
		return
	}

	steps := []string{
		`DELETE FROM session_tasks WHERE session_id IN (SELECT id FROM sessions WHERE child_id = ?)`,
		`DELETE FROM sessions        WHERE child_id = ?`,
		`DELETE FROM weekly_streaks  WHERE child_id = ?`,
		`DELETE FROM wheel_options   WHERE child_id = ?`,
		`DELETE FROM fcm_tokens      WHERE child_id = ?`,
		`DELETE FROM user_stats      WHERE child_id = ?`,
		`DELETE FROM children        WHERE id = ?`,
	}
	for _, q := range steps {
		if _, err := tx.Exec(q, id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

type adminSummary struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

// ListAdmins returns all admins (id + username only, no password hash).
// GET /api/admin/admins
func ListAdmins(c *gin.Context) {
	rows, err := db.DB.Query(`SELECT id, username FROM admins ORDER BY id`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer rows.Close()

	var admins []adminSummary
	for rows.Next() {
		var a adminSummary
		if err := rows.Scan(&a.ID, &a.Username); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		admins = append(admins, a)
	}
	if admins == nil {
		admins = []adminSummary{}
	}
	c.JSON(http.StatusOK, admins)
}

type createAdminRequest struct {
	Username string `json:"username" binding:"required,max=100"`
	Password string `json:"password" binding:"required,min=8,max=100"`
}

// CreateAdmin creates a new admin account.
// POST /api/admin/admins
func CreateAdmin(c *gin.Context) {
	var req createAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "wachtwoord hashing mislukt"})
		return
	}

	res, err := db.DB.Exec(
		`INSERT INTO admins (username, password_hash) VALUES (?, ?)`,
		req.Username, string(hash),
	)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "gebruikersnaam bestaat al"})
		return
	}

	id, _ := res.LastInsertId()
	c.JSON(http.StatusCreated, adminSummary{ID: id, Username: req.Username})
}

// DeleteAdmin removes an admin account and their refresh tokens.
// Cannot delete yourself.
// DELETE /api/admin/admins/:id
func DeleteAdmin(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ongeldig id"})
		return
	}

	selfID := c.MustGet("admin_id").(int64)
	if id == selfID {
		c.JSON(http.StatusForbidden, gin.H{"error": "je kunt je eigen account niet verwijderen"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec(`DELETE FROM refresh_tokens WHERE admin_id = ?`, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	res, err := tx.Exec(`DELETE FROM admins WHERE id = ?`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "beheerder niet gevonden"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

type updateAdminRequest struct {
	Username string `json:"username" binding:"max=100"`
	Password string `json:"password" binding:"omitempty,min=8,max=100"`
}

// UpdateAdmin changes an admin's username and/or password.
// PATCH /api/admin/admins/:id
func UpdateAdmin(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ongeldig id"})
		return
	}

	var req updateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Username == "" && req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "geen wijzigingen opgegeven"})
		return
	}

	var exists int
	if err := db.DB.QueryRow(`SELECT 1 FROM admins WHERE id = ?`, id).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "beheerder niet gevonden"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		}
		return
	}

	if req.Username != "" {
		if _, err := db.DB.Exec(`UPDATE admins SET username = ? WHERE id = ?`, req.Username, id); err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "gebruikersnaam bestaat al"})
			return
		}
	}

	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "wachtwoord hashing mislukt"})
			return
		}
		if _, err := db.DB.Exec(`UPDATE admins SET password_hash = ? WHERE id = ?`, string(hash), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
