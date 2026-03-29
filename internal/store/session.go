package store

import (
	"context"
	"fmt"
	"time"
)

// Session represents a row in the sessions table.
type Session struct {
	ID          string
	UserID      string
	NotionToken string // AES-256-GCM encrypted, base64-encoded
	ExpiresAt   time.Time
	CreatedAt   string
}

// InsertSession persists a new session.
func (d *DB) InsertSession(ctx context.Context, s Session) error {
	_, err := d.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, notion_token, expires_at)
		 VALUES (?, ?, ?, ?)`,
		s.ID, s.UserID, s.NotionToken, s.ExpiresAt.UTC().Format(time.RFC3339),
	)
	return err
}

// GetSession returns a session by ID.
// Returns ErrNotFound if not found, ErrConflict if expired.
func (d *DB) GetSession(ctx context.Context, id string) (*Session, error) {
	row := d.QueryRowContext(ctx,
		`SELECT id, user_id, notion_token, expires_at, created_at
		 FROM sessions WHERE id = ?`, id)

	var s Session
	var expiresStr string
	if err := row.Scan(&s.ID, &s.UserID, &s.NotionToken, &expiresStr, &s.CreatedAt); err != nil {
		if isNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan session: %w", err)
	}

	t, err := time.Parse(time.RFC3339, expiresStr)
	if err != nil {
		return nil, fmt.Errorf("parse expires_at: %w", err)
	}
	s.ExpiresAt = t
	return &s, nil
}

// DeleteSession removes a session (logout).
func (d *DB) DeleteSession(ctx context.Context, id string) error {
	_, err := d.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	return err
}

// DeleteExpiredSessions removes all sessions that have passed their expiry.
func (d *DB) DeleteExpiredSessions(ctx context.Context) error {
	_, err := d.ExecContext(ctx,
		`DELETE FROM sessions WHERE expires_at < ?`,
		time.Now().UTC().Format(time.RFC3339),
	)
	return err
}
