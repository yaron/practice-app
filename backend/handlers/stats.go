package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"violin-quest-api/db"
	"violin-quest-api/models"

	"github.com/gin-gonic/gin"
)

// GetStats returns progression stats for a child, including the current week's streak.
// GET /api/stats?child_id=1
func GetStats(c *gin.Context) {
	childID, err := strconv.ParseInt(c.Query("child_id"), 10, 64)
	if err != nil || childID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "child_id is required"})
		return
	}

	var stats models.UserStats
	err = db.DB.QueryRow(
		`SELECT id, child_id, total_points, current_level, experience_points, shield_count
		 FROM user_stats WHERE child_id = ?`,
		childID,
	).Scan(&stats.ID, &stats.ChildID, &stats.TotalPoints, &stats.CurrentLevel,
		&stats.ExperiencePoints, &stats.ShieldCount)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no stats for child"})
		return
	}

	year, week := time.Now().ISOWeek()
	yearWeek := isoWeekKey(year, week)

	var streak models.WeeklyStreak
	streak.YearWeek = yearWeek
	var milestoneInt int
	db.DB.QueryRow( //nolint:errcheck — returns zero values on no row, which is correct
		`SELECT session_count, milestone_reached FROM weekly_streaks
		 WHERE child_id = ? AND year_week = ?`,
		childID, yearWeek,
	).Scan(&streak.SessionCount, &milestoneInt)
	streak.MilestoneReached = milestoneInt == 1

	c.JSON(http.StatusOK, models.StatsResponse{
		CurrentLevel:     stats.CurrentLevel,
		ExperiencePoints: stats.ExperiencePoints,
		TotalPoints:      stats.TotalPoints,
		ShieldCount:      stats.ShieldCount,
		WeekSessionCount: streak.SessionCount,
		MilestoneReached: streak.MilestoneReached,
		YearWeek:         yearWeek,
	})
}

func isoWeekKey(year, week int) string {
	return fmt.Sprintf("%d-W%02d", year, week)
}
