package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

func Open(ctx context.Context) (*sql.DB, error) {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".sussout")
	os.MkdirAll(dir, 0755)
	dsn := filepath.Join(dir, "sussout.db") + "?_loc=auto"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.ExecContext(ctx, "PRAGMA journal_mode = WAL")
	db.ExecContext(ctx, "PRAGMA foreign_keys = ON")

	if err := migrate(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return db, nil
}

func migrate(ctx context.Context, db *sql.DB) error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id INTEGER REFERENCES sessions(id) ON DELETE CASCADE,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS assumptions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id INTEGER REFERENCES sessions(id) ON DELETE CASCADE,
			content TEXT NOT NULL,
			status TEXT DEFAULT 'identified',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS decisions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id INTEGER REFERENCES sessions(id) ON DELETE CASCADE,
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, ddl := range tables {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("create table: %w", err)
		}
	}

	if err := fixSQLiteColumnTypes(ctx, db); err != nil {
		return fmt.Errorf("fix column types: %w", err)
	}

	return nil
}

func fixSQLiteColumnTypes(ctx context.Context, db *sql.DB) error {
	tables := []string{"sessions", "messages", "assumptions", "decisions"}
	for _, table := range tables {
		var colType string
		err := db.QueryRowContext(ctx,
			`SELECT type FROM pragma_table_info(?) WHERE name = 'created_at'`, table,
		).Scan(&colType)
		if err != nil {
			continue
		}
		if colType == "TEXT" {
			recreateSQLiteTable(ctx, db, table)
		}
	}
	return nil
}

func recreateSQLiteTable(ctx context.Context, db *sql.DB, table string) {
	var sql string
	db.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&sql)
	sql = strings.ReplaceAll(sql, "TEXT DEFAULT CURRENT_TIMESTAMP", "DATETIME DEFAULT CURRENT_TIMESTAMP")
	sql = strings.ReplaceAll(sql, " TEXT ,", " DATETIME,")

	db.ExecContext(ctx, "PRAGMA foreign_keys = OFF")
	db.ExecContext(ctx, fmt.Sprintf("CREATE TABLE %s_new AS SELECT * FROM %s", table, table))
	db.ExecContext(ctx, fmt.Sprintf("DROP TABLE %s", table))
	db.ExecContext(ctx, sql)
	db.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s SELECT * FROM %s_new", table, table))
	db.ExecContext(ctx, fmt.Sprintf("DROP TABLE %s_new", table))
	db.ExecContext(ctx, "PRAGMA foreign_keys = ON")
}
