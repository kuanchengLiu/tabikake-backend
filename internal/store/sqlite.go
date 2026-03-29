package store

import (
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"
)

// ErrNotFound is returned when a requested row does not exist.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned when a unique constraint is violated.
var ErrConflict = errors.New("already exists")

// DB wraps the SQLite connection pool.
type DB struct {
	*sql.DB
}

// New opens (or creates) the SQLite database at path and runs migrations.
func New(path string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	d := &DB{sqlDB}
	if err := d.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return d, nil
}

func (d *DB) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id         TEXT PRIMARY KEY,
			name       TEXT NOT NULL,
			avatar_url TEXT,
			created_at TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id           TEXT PRIMARY KEY,
			user_id      TEXT NOT NULL REFERENCES users(id),
			notion_token TEXT NOT NULL,
			expires_at   TEXT NOT NULL,
			created_at   TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS trips (
			id             TEXT PRIMARY KEY,
			name           TEXT NOT NULL,
			owner_id       TEXT NOT NULL REFERENCES users(id),
			notion_page_id TEXT NOT NULL,
			notion_db_id   TEXT NOT NULL,
			budget_jpy     INTEGER DEFAULT 0,
			budget_suica   INTEGER DEFAULT 0,
			start_date     TEXT,
			end_date       TEXT,
			invite_code    TEXT UNIQUE NOT NULL,
			created_at     TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS members (
			id        TEXT PRIMARY KEY,
			trip_id   TEXT NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
			user_id   TEXT NOT NULL REFERENCES users(id),
			is_owner  INTEGER NOT NULL DEFAULT 0,
			joined_at TEXT DEFAULT (datetime('now')),
			UNIQUE(trip_id, user_id)
		)`,
	}
	for _, stmt := range stmts {
		if _, err := d.Exec(stmt); err != nil {
			return fmt.Errorf("%w", err)
		}
	}
	return nil
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func isNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
