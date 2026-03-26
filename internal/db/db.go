package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/yourname/tabikake/internal/model"
)

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("not found")

// DB wraps the SQLite connection.
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
	_, err := d.Exec(`
		CREATE TABLE IF NOT EXISTS trips (
			id              TEXT PRIMARY KEY,
			name            TEXT NOT NULL,
			start_date      TEXT,
			end_date        TEXT,
			notion_page_id  TEXT NOT NULL,
			notion_db_id    TEXT NOT NULL,
			created_at      TEXT DEFAULT (datetime('now'))
		)
	`)
	return err
}

// InsertTrip saves a new trip to SQLite.
func (d *DB) InsertTrip(ctx context.Context, trip model.Trip) error {
	_, err := d.ExecContext(ctx,
		`INSERT INTO trips (id, name, start_date, end_date, notion_page_id, notion_db_id)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		trip.ID, trip.Name, trip.StartDate, trip.EndDate, trip.NotionPageID, trip.NotionDbID,
	)
	return err
}

// GetTrip returns a single trip by ID.
func (d *DB) GetTrip(ctx context.Context, id string) (*model.Trip, error) {
	row := d.QueryRowContext(ctx,
		`SELECT id, name, start_date, end_date, notion_page_id, notion_db_id, created_at
		 FROM trips WHERE id = ?`, id)
	return scanTrip(row)
}

// ListTrips returns all trips ordered by creation time descending.
func (d *DB) ListTrips(ctx context.Context) ([]model.Trip, error) {
	rows, err := d.QueryContext(ctx,
		`SELECT id, name, start_date, end_date, notion_page_id, notion_db_id, created_at
		 FROM trips ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trips []model.Trip
	for rows.Next() {
		trip, err := scanTrip(rows)
		if err != nil {
			return nil, err
		}
		trips = append(trips, *trip)
	}
	return trips, rows.Err()
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanTrip(s scanner) (*model.Trip, error) {
	var t model.Trip
	err := s.Scan(&t.ID, &t.Name, &t.StartDate, &t.EndDate, &t.NotionPageID, &t.NotionDbID, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan trip: %w", err)
	}
	return &t, nil
}
