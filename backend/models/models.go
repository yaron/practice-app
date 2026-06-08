package models

type Child struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Timezone string `json:"timezone"`
}

type WheelOption struct {
	ID      int64  `json:"id"`
	ChildID int64  `json:"child_id"`
	Text    string `json:"text"`
	IsBonus bool   `json:"is_bonus"`
}

type Session struct {
	ID             int64  `json:"id"`
	ChildID        int64  `json:"child_id"`
	Date           string `json:"date"`
	TasksCompleted int    `json:"tasks_completed"`
	Status         string `json:"status"`
	RejectionNote  string `json:"rejection_note,omitempty"`
	IsFirstOfDay   bool   `json:"is_first_of_day"`
	CreatedAt      string `json:"created_at"`
}

type UserStats struct {
	ID               int64 `json:"id"`
	ChildID          int64 `json:"child_id"`
	TotalPoints      int   `json:"total_points"`
	CurrentLevel     int   `json:"current_level"`
	ExperiencePoints int   `json:"experience_points"`
	ShieldCount      int   `json:"shield_count"`
}

type WeeklyStreak struct {
	ChildID          int64  `json:"child_id"`
	YearWeek         string `json:"year_week"`
	SessionCount     int    `json:"session_count"`
	MilestoneReached bool   `json:"milestone_reached"`
}

type SessionWithTasks struct {
	ID             int64    `json:"id"`
	ChildID        int64    `json:"child_id"`
	ChildName      string   `json:"child_name"`
	Date           string   `json:"date"`
	TasksCompleted int      `json:"tasks_completed"`
	Status         string   `json:"status"`
	RejectionNote  string   `json:"rejection_note,omitempty"`
	Tasks          []string `json:"tasks"`
	CreatedAt      string   `json:"created_at"`
}

type Admin struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
}

type PendingSession struct {
	ID             int64  `json:"id"`
	ChildID        int64  `json:"child_id"`
	ChildName      string `json:"child_name"`
	Date           string `json:"date"`
	TasksCompleted int    `json:"tasks_completed"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at"`
}

type StatsResponse struct {
	CurrentLevel     int    `json:"current_level"`
	ExperiencePoints int    `json:"experience_points"`
	TotalPoints      int    `json:"total_points"`
	ShieldCount      int    `json:"shield_count"`
	WeekSessionCount int    `json:"week_session_count"`
	MilestoneReached bool   `json:"milestone_reached"`
	YearWeek         string `json:"year_week"`
}
