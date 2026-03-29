package store

import (
	"context"
	"fmt"

	"github.com/yourname/tabikake/internal/model"
)

// UpsertUser inserts a new user or updates name/avatar_url if the user already exists.
func (d *DB) UpsertUser(ctx context.Context, user model.User) error {
	_, err := d.ExecContext(ctx,
		`INSERT INTO users (id, name, avatar_url)
		 VALUES (?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   name       = excluded.name,
		   avatar_url = excluded.avatar_url`,
		user.ID, user.Name, user.AvatarURL,
	)
	return err
}

// GetUser returns a user by ID.
func (d *DB) GetUser(ctx context.Context, id string) (*model.User, error) {
	row := d.QueryRowContext(ctx,
		`SELECT id, name, avatar_url, created_at FROM users WHERE id = ?`, id)
	var u model.User
	if err := row.Scan(&u.ID, &u.Name, &u.AvatarURL, &u.CreatedAt); err != nil {
		if isNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return &u, nil
}

// GetUsersByIDs returns a map of user_id → User for the given IDs.
func (d *DB) GetUsersByIDs(ctx context.Context, ids []string) (map[string]model.User, error) {
	result := make(map[string]model.User, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	// Query each user individually to avoid building dynamic SQL.
	for _, id := range ids {
		u, err := d.GetUser(ctx, id)
		if err != nil {
			continue // skip unknown IDs
		}
		result[id] = *u
	}
	return result, nil
}
