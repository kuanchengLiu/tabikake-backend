package store

import (
	"context"
	"fmt"

	"github.com/yourname/tabikake/internal/model"
)

// InsertMember adds a user to a trip.
// Returns ErrConflict if the user is already a member.
func (d *DB) InsertMember(ctx context.Context, m model.Member) error {
	_, err := d.ExecContext(ctx,
		`INSERT INTO members (id, trip_id, user_id, is_owner) VALUES (?, ?, ?, ?)`,
		m.ID, m.TripID, m.UserID, boolToInt(m.IsOwner),
	)
	if err != nil && isSQLiteUnique(err) {
		return ErrConflict
	}
	return err
}

// GetMemberByUser returns the member record for a specific user in a trip.
func (d *DB) GetMemberByUser(ctx context.Context, tripID, userID string) (*model.Member, error) {
	row := d.QueryRowContext(ctx,
		`SELECT id, trip_id, user_id, is_owner, joined_at
		 FROM members WHERE trip_id = ? AND user_id = ?`, tripID, userID)
	return scanMember(row)
}

// ListMembers returns all members for a trip ordered by join date.
func (d *DB) ListMembers(ctx context.Context, tripID string) ([]model.Member, error) {
	rows, err := d.QueryContext(ctx,
		`SELECT id, trip_id, user_id, is_owner, joined_at
		 FROM members WHERE trip_id = ? ORDER BY joined_at ASC`, tripID)
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

// DeleteMember removes a user from a trip by user_id.
func (d *DB) DeleteMember(ctx context.Context, tripID, userID string) error {
	res, err := d.ExecContext(ctx,
		`DELETE FROM members WHERE trip_id = ? AND user_id = ?`, tripID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// IsMember returns true if userID is a member of tripID.
func (d *DB) IsMember(ctx context.Context, tripID, userID string) (bool, error) {
	var count int
	err := d.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM members WHERE trip_id = ? AND user_id = ?`, tripID, userID).
		Scan(&count)
	return count > 0, err
}

// IsOwner returns true if userID is the owner of tripID.
func (d *DB) IsOwner(ctx context.Context, tripID, userID string) (bool, error) {
	var count int
	err := d.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM members WHERE trip_id = ? AND user_id = ? AND is_owner = 1`,
		tripID, userID).Scan(&count)
	return count > 0, err
}

// CountMembers returns the number of members in a trip.
func (d *DB) CountMembers(ctx context.Context, tripID string) (int, error) {
	var count int
	err := d.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM members WHERE trip_id = ?`, tripID).Scan(&count)
	return count, err
}

// GetTripOwnerName returns the name of the trip owner by looking up the owner's user record.
func (d *DB) GetTripOwnerName(ctx context.Context, tripID string) (string, error) {
	var name string
	err := d.QueryRowContext(ctx,
		`SELECT u.name FROM users u
		 JOIN members m ON u.id = m.user_id
		 WHERE m.trip_id = ? AND m.is_owner = 1 LIMIT 1`, tripID).Scan(&name)
	if isNoRows(err) {
		return "", ErrNotFound
	}
	return name, err
}

func scanMember(s scanner) (*model.Member, error) {
	var m model.Member
	var isOwner int
	err := s.Scan(&m.ID, &m.TripID, &m.UserID, &isOwner, &m.JoinedAt)
	if isNoRows(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan member: %w", err)
	}
	m.IsOwner = isOwner == 1
	return &m, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// isSQLiteUnique detects SQLite UNIQUE constraint violations.
func isSQLiteUnique(err error) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), "UNIQUE constraint failed")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
