package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"violin-quest-api/db"
	"violin-quest-api/models"

	"github.com/gin-gonic/gin"
)

// GetOptions returns all wheel options for a given child.
// GET /api/options?child_id=1
func GetOptions(c *gin.Context) {
	childID, err := strconv.ParseInt(c.Query("child_id"), 10, 64)
	if err != nil || childID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "child_id is required"})
		return
	}

	rows, err := db.DB.Query(
		`SELECT id, child_id, text, short_text, is_bonus FROM wheel_options WHERE child_id = ? ORDER BY id`,
		childID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer rows.Close()

	options := []models.WheelOption{}
	for rows.Next() {
		var o models.WheelOption
		var isBonus int
		if err := rows.Scan(&o.ID, &o.ChildID, &o.Text, &o.ShortText, &isBonus); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		o.IsBonus = isBonus == 1
		options = append(options, o)
	}

	c.JSON(http.StatusOK, options)
}

type createOptionRequest struct {
	ChildID   int64  `json:"child_id" binding:"required"`
	Text      string `json:"text" binding:"required,max=200"`
	ShortText string `json:"short_text" binding:"required,max=50"`
	IsBonus   bool   `json:"is_bonus"`
}

// CreateOption adds a new wheel option for a child.
// POST /api/admin/options
func CreateOption(c *gin.Context) {
	var req createOptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var exists int
	if err := db.DB.QueryRow(`SELECT 1 FROM children WHERE id = ?`, req.ChildID).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "kind niet gevonden"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		}
		return
	}

	isBonus := 0
	if req.IsBonus {
		isBonus = 1
	}
	res, err := db.DB.Exec(
		`INSERT INTO wheel_options (child_id, text, short_text, is_bonus) VALUES (?, ?, ?, ?)`,
		req.ChildID, req.Text, req.ShortText, isBonus,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	id, _ := res.LastInsertId()
	c.JSON(http.StatusCreated, models.WheelOption{
		ID: id, ChildID: req.ChildID, Text: req.Text, ShortText: req.ShortText, IsBonus: req.IsBonus,
	})
}

type updateOptionRequest struct {
	Text      string `json:"text" binding:"required,max=200"`
	ShortText string `json:"short_text" binding:"required,max=50"`
	IsBonus   bool   `json:"is_bonus"`
}

// UpdateOption replaces text, short_text, and is_bonus for a wheel option.
// PATCH /api/admin/options/:id
func UpdateOption(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ongeldig id"})
		return
	}

	var req updateOptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var childID int64
	if err := db.DB.QueryRow(`SELECT child_id FROM wheel_options WHERE id = ?`, id).Scan(&childID); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "optie niet gevonden"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		}
		return
	}

	isBonus := 0
	if req.IsBonus {
		isBonus = 1
	}
	if _, err := db.DB.Exec(
		`UPDATE wheel_options SET text = ?, short_text = ?, is_bonus = ? WHERE id = ?`,
		req.Text, req.ShortText, isBonus, id,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.JSON(http.StatusOK, models.WheelOption{
		ID: id, ChildID: childID, Text: req.Text, ShortText: req.ShortText, IsBonus: req.IsBonus,
	})
}

// DeleteOption removes a wheel option.
// DELETE /api/admin/options/:id
func DeleteOption(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ongeldig id"})
		return
	}

	res, err := db.DB.Exec(`DELETE FROM wheel_options WHERE id = ?`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "optie niet gevonden"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}
