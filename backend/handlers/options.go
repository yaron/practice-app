package handlers

import (
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
