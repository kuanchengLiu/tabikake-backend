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
			invite_code     TEXT UNIQUE,
			created_at      TEXT DEFAULT (datetime('now'))
		)
	`)
	if err != nil {
		return err
	}

	// Add invite_code column to existing tables that predate this migration.
	_, _ = d.Exec(`ALTER TABLE trips ADD COLUMN invite_code TEXT`)

	_, err = d.Exec(`
		CREATE TABLE IF NOT EXISTS members (
			id           TEXT PRIMARY KEY,
			trip_id      TEXT NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
			name         TEXT NOT NULL,
			avatar_color TEXT NOT NULL DEFAULT '#F59E0B',
			is_owner     INTEGER NOT NULL DEFAULT 0,
			created_at   TEXT DEFAULT (datetime('now'))
		)
	`)
	return err
}

// --- Trip CRUD ---

// InsertTrip saves a new trip to SQLite.
func (d *DB) InsertTrip(ctx context.Context, trip model.Trip) error {
	_, err := d.ExecContext(ctx,
		`INSERT INTO trips (id, name, start_date, end_date, notion_page_id, notion_db_id, invite_code)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		trip.ID, trip.Name, trip.StartDate, trip.EndDate, trip.NotionPageID, trip.NotionDbID, trip.InviteCode,
	)
	return err
}

// GetTrip returns a single trip by ID.
func (d *DB) GetTrip(ctx context.Context, id string) (*model.Trip, error) {
	row := d.QueryRowContext(ctx,
		`SELECT id, name, start_date, end_date, notion_page_id, notion_db_id, COALESCE(invite_code,''), created_at
		 FROM trips WHERE id = ?`, id)
	return scanTrip(row)
}

// GetTripByInviteCode returns a trip by its invite code.
func (d *DB) GetTripByInviteCode(ctx context.Context, code string) (*model.Trip, error) {
	row := d.QueryRowContext(ctx,
		`SELECT id, name, start_date, end_date, notion_page_id, notion_db_id, COALESCE(invite_code,''), created_at
		 FROM trips WHERE invite_code = ?`, code)
	return scanTrip(row)
}

// ListTrips returns all trips ordered by creation time descending.
func (d *DB) ListTrips(ctx context.Context) ([]model.Trip, error) {
	rows, err := d.QueryContext(ctx,
		`SELECT id, name, start_date, end_date, notion_page_id, notion_db_id, COALESCE(invite_code,''), created_at
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

// --- Member CRUD ---

// InsertMember saves a new member to SQLite.
func (d *DB) InsertMember(ctx context.Context, m model.Member) error {
	isOwner := 0
	if m.IsOwner {
		isOwner = 1
	}
	_, err := d.ExecContext(ctx,
		`INSERT INTO members (id, trip_id, name, avatar_color, is_owner)
		 VALUES (?, ?, ?, ?, ?)`,
		m.ID, m.TripID, m.Name, m.AvatarColor, isOwner,
	)
	return err
}

// GetMember returns a single member by ID.
func (d *DB) GetMember(ctx context.Context, id string) (*model.Member, error) {
	row := d.QueryRowContext(ctx,
		`SELECT id, trip_id, name, avatar_color, is_owner, created_at
		 FROM members WHERE id = ?`, id)
	return scanMember(row)
}

// ListMembers returns all members for a trip.
func (d *DB) ListMembers(ctx context.Context, tripID string) ([]model.Member, error) {
	rows, err := d.QueryContext(ctx,
		`SELECT id, trip_id, name, avatar_color, is_owner, created_at
		 FROM members WHERE trip_id = ? ORDER BY created_at ASC`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []model.Member
	for rows.Next() {
		m, err := scanMember(rows)
		if err != nil {
			return nil, err
		}
		members = append(members, *m)
	}
	return members, rows.Err()
}

// IsMember returns true if memberID belongs to the given trip.
func (d *DB) IsMember(ctx context.Context, tripID, memberID string) (bool, error) {
	var count int
	err := d.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM members WHERE trip_id = ? AND id = ?`, tripID, memberID).Scan(&count)
	return count > 0, err
}

// GetTripOwner returns the owner member of a trip.
func (d *DB) GetTripOwner(ctx context.Context, tripID string) (*model.Member, error) {
	row := d.QueryRowContext(ctx,
		`SELECT id, trip_id, name, avatar_color, is_owner, created_at
		 FROM members WHERE trip_id = ? AND is_owner = 1 LIMIT 1`, tripID)
	return scanMember(row)
}

// CountMembers returns the number of members in a trip.
func (d *DB) CountMembers(ctx context.Context, tripID string) (int, error) {
	var count int
	err := d.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM members WHERE trip_id = ?`, tripID).Scan(&count)
	return count, err
}

// DeleteMember removes a member by ID.
func (d *DB) DeleteMember(ctx context.Context, id string) error {
	res, err := d.ExecContext(ctx, `DELETE FROM members WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// --- scanner helpers ---

type scanner interface {
	Scan(dest ...any) error
}

func scanTrip(s scanner) (*model.Trip, error) {
	var t model.Trip
	err := s.Scan(&t.ID, &t.Name, &t.StartDate, &t.EndDate, &t.NotionPageID, &t.NotionDbID, &t.InviteCode, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan trip: %w", err)
	}
	return &t, nil
}

func scanMember(s scanner) (*model.Member, error) {
	var m model.Member
	var isOwner int
	err := s.Scan(&m.ID, &m.TripID, &m.Name, &m.AvatarColor, &isOwner, &m.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan member: %w", err)
	}
	m.IsOwner = isOwner == 1
	return &m, nil
}
