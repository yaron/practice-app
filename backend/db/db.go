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
