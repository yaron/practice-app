package db

import "log"

func Migrate() {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS admins (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			username      TEXT    UNIQUE NOT NULL,
			password_hash TEXT    NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS children (
			id       INTEGER PRIMARY KEY AUTOINCREMENT,
			name     TEXT    NOT NULL,
			timezone TEXT    NOT NULL DEFAULT 'Europe/Amsterdam'
		)`,
		`CREATE TABLE IF NOT EXISTS user_stats (
			id                INTEGER PRIMARY KEY AUTOINCREMENT,
			child_id          INTEGER NOT NULL REFERENCES children(id),
			total_points      INTEGER DEFAULT 0,
			current_level     INTEGER DEFAULT 1,
			experience_points INTEGER DEFAULT 0,
			shield_count      INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			child_id        INTEGER NOT NULL REFERENCES children(id),
			date            TEXT    NOT NULL,
			tasks_completed INTEGER NOT NULL,
			status          TEXT    NOT NULL DEFAULT 'PENDING',
			rejection_note  TEXT,
			is_first_of_day INTEGER DEFAULT 0,
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_child_date_status
			ON sessions(child_id, date, status)`,
		`CREATE TABLE IF NOT EXISTS weekly_streaks (
			child_id          INTEGER NOT NULL REFERENCES children(id),
			year_week         TEXT    NOT NULL,
			session_count     INTEGER DEFAULT 0,
			milestone_reached INTEGER DEFAULT 0,
			PRIMARY KEY (child_id, year_week)
		)`,
		`CREATE TABLE IF NOT EXISTS wheel_options (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			child_id   INTEGER NOT NULL REFERENCES children(id),
			text       TEXT    NOT NULL,
			short_text TEXT    NOT NULL DEFAULT '',
			is_bonus   INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS fcm_tokens (
			token      TEXT     PRIMARY KEY,
			child_id   INTEGER  REFERENCES children(id),
			role       TEXT     NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS store_items (
			id   INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT    NOT NULL,
			type TEXT    NOT NULL,
			cost INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS refresh_tokens (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			admin_id   INTEGER NOT NULL REFERENCES admins(id),
			token_hash TEXT    NOT NULL UNIQUE,
			expires_at DATETIME NOT NULL,
			revoked    INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS session_tasks (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id INTEGER NOT NULL REFERENCES sessions(id),
			task_text  TEXT    NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_session_tasks_session
			ON session_tasks(session_id)`,
	}

	for _, s := range statements {
		if _, err := DB.Exec(s); err != nil {
			log.Fatalf("migrate: %v\nstatement: %s", err, s)
		}
	}

	log.Println("db: schema up to date")
}
