package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

func Open(ctx context.Context, connString string) (*sql.DB, string, error) {
	var driver, dsn string

	if connString == "" {
		home, _ := os.UserHomeDir()
		dir := filepath.Join(home, ".sussout")
		os.MkdirAll(dir, 0755)
		dsn = filepath.Join(dir, "sussout.db")
		driver = "sqlite"
	} else if strings.HasPrefix(connString, "postgres://") || strings.HasPrefix(connString, "postgresql://") {
		dsn = connString
		if !strings.Contains(dsn, "sslmode=") {
			if strings.Contains(dsn, "?") {
				dsn += "&"
			} else {
				dsn += "?"
			}
			dsn += "sslmode=disable"
		}
		driver = "postgres"
	} else {
		dsn = connString
		driver = "sqlite"
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, "", fmt.Errorf("failed to ping database: %w", err)
	}

	if driver == "sqlite" {
		db.ExecContext(ctx, "PRAGMA journal_mode = WAL")
		db.ExecContext(ctx, "PRAGMA foreign_keys = ON")
	}

	if err := migrate(ctx, db, driver); err != nil {
		db.Close()
		return nil, "", fmt.Errorf("migration failed: %w", err)
	}

	return db, driver, nil
}

func migrate(ctx context.Context, db *sql.DB, driver string) error {
	pg := driver == "postgres"
	serial := "SERIAL"
	timestamp := "TIMESTAMP WITH TIME ZONE"
	if !pg {
		serial = "INTEGER"
		timestamp = "TEXT"
	}

	tables := []string{
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS sessions (
			id %s PRIMARY KEY %s,
			title TEXT,
			created_at %s DEFAULT CURRENT_TIMESTAMP,
			updated_at %s DEFAULT CURRENT_TIMESTAMP
		)`, serial, autoinc(!pg), timestamp, timestamp),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS messages (
			id %s PRIMARY KEY %s,
			session_id INTEGER REFERENCES sessions(id) ON DELETE CASCADE,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at %s DEFAULT CURRENT_TIMESTAMP
		)`, serial, autoinc(!pg), timestamp),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS assumptions (
			id %s PRIMARY KEY %s,
			session_id INTEGER REFERENCES sessions(id) ON DELETE CASCADE,
			content TEXT NOT NULL,
			status TEXT DEFAULT 'identified',
			created_at %s DEFAULT CURRENT_TIMESTAMP
		)`, serial, autoinc(!pg), timestamp),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS decisions (
			id %s PRIMARY KEY %s,
			session_id INTEGER REFERENCES sessions(id) ON DELETE CASCADE,
			content TEXT NOT NULL,
			created_at %s DEFAULT CURRENT_TIMESTAMP
		)`, serial, autoinc(!pg), timestamp),
	}

	for _, ddl := range tables {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("create table: %w", err)
		}
	}

	return nil
}

func autoinc(sqlite bool) string {
	if sqlite {
		return "AUTOINCREMENT"
	}
	return ""
}