package db

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Open() {
	path := os.Getenv("DB_PATH")
	if path == "" {
		path = "./violin-quest.db"
	}

	var err error
	DB, err = sql.Open("sqlite3", path)
	if err != nil {
		log.Fatalf("db: failed to open %s: %v", path, err)
	}

	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA foreign_keys=ON;",
	}
	for _, p := range pragmas {
		if _, err := DB.Exec(p); err != nil {
			log.Fatalf("db: pragma failed (%s): %v", p, err)
		}
	}

	log.Printf("db: opened %s", path)
}

func GetAdminTokens() []string {
	rows, err := DB.Query(`SELECT token FROM fcm_tokens WHERE role = 'ADMIN'`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var tokens []string
	for rows.Next() {
		var t string
		rows.Scan(&t) //nolint:errcheck
		tokens = append(tokens, t)
	}
	return tokens
}

func GetChildTokens(childID int64) []string {
	rows, err := DB.Query(`SELECT token FROM fcm_tokens WHERE role = 'CHILD' AND child_id = ?`, childID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var tokens []string
	for rows.Next() {
		var t string
		rows.Scan(&t) //nolint:errcheck
		tokens = append(tokens, t)
	}
	return tokens
}

func DeleteFCMToken(token string) {
	DB.Exec(`DELETE FROM fcm_tokens WHERE token = ?`, token) //nolint:errcheck
}
