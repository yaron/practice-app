package db

import (
	"log"

	"golang.org/x/crypto/bcrypt"
)

// DefaultWheelOption is a single default wheel segment definition.
type DefaultWheelOption struct {
	Text      string
	ShortText string
	IsBonus   int
}

// DefaultWheelOptions are the default tasks assigned to every new child.
// Text is the full task name; ShortText is the abbreviated canvas label.
// IsBonus marks segments that trigger the confetti animation (Phase 7).
var DefaultWheelOptions = []DefaultWheelOption{
	{"Speel een liedje op 1 been", "1 Been 1️⃣", 0},
	{"Speel een liedje boos", "Boos 😠", 0},
	{"Speel een liedje blij", "Blij 😁", 0},
	{"Speel een liedje terwijl je stampt", "Stampend 🦵", 0},
	{"Speel een liedje gewoon", "Gewoon 🤷", 0},
	{"Speel een liedje gewoon", "Gewoontjes 👍", 0},
	{"Speel een liedje terwijl iemand je afleid", "Afleiding 😝", 0},
}

// Seed inserts initial data if the database is empty (no children yet).
// Safe to call on every startup — it is a no-op when data already exists.
func Seed() {
	var count int
	if err := DB.QueryRow("SELECT COUNT(*) FROM children").Scan(&count); err != nil {
		log.Fatalf("seed: count children: %v", err)
	}
	if count > 0 {
		return
	}

	tx, err := DB.Begin()
	if err != nil {
		log.Fatalf("seed: begin tx: %v", err)
	}
	defer tx.Rollback()

	// Child
	res, err := tx.Exec(
		`INSERT INTO children (name, timezone) VALUES (?, ?)`,
		"Speler", "Europe/Amsterdam",
	)
	if err != nil {
		log.Fatalf("seed: insert child: %v", err)
	}
	childID, _ := res.LastInsertId()

	// user_stats row for the child
	if _, err := tx.Exec(
		`INSERT INTO user_stats (child_id) VALUES (?)`, childID,
	); err != nil {
		log.Fatalf("seed: insert user_stats: %v", err)
	}

	// Wheel options — scoped to the seeded child
	for _, opt := range DefaultWheelOptions {
		if _, err := tx.Exec(
			`INSERT INTO wheel_options (child_id, text, short_text, is_bonus) VALUES (?, ?, ?, ?)`,
			childID, opt.Text, opt.ShortText, opt.IsBonus,
		); err != nil {
			log.Fatalf("seed: insert wheel_option: %v", err)
		}
	}

	// Admin account (password should be changed on first login)
	hash, err := bcrypt.GenerateFromPassword([]byte("changeme"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("seed: bcrypt: %v", err)
	}
	if _, err := tx.Exec(
		`INSERT INTO admins (username, password_hash) VALUES (?, ?)`,
		"admin", string(hash),
	); err != nil {
		log.Fatalf("seed: insert admin: %v", err)
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("seed: commit: %v", err)
	}

	log.Printf("db: seeded child_id=%d with %d wheel options", childID, len(DefaultWheelOptions))
}
