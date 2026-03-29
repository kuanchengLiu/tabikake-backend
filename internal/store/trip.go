package store

import (
	"context"
	"fmt"

	"github.com/yourname/tabikake/internal/model"
)

// InsertTrip persists a new trip.
func (d *DB) InsertTrip(ctx context.Context, t model.Trip) error {
	_, err := d.ExecContext(ctx,
		`INSERT INTO trips
		   (id, name, owner_id, notion_page_id, notion_db_id,
		    budget_jpy, budget_suica, start_date, end_date, invite_code)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Name, t.OwnerID, t.NotionPageID, t.NotionDbID,
		t.BudgetJPY, t.BudgetSuica, t.StartDate, t.EndDate, t.InviteCode,
	)
	return err
}

// GetTrip returns a trip by ID.
func (d *DB) GetTrip(ctx context.Context, id string) (*model.Trip, error) {
	row := d.QueryRowContext(ctx,
		`SELECT id, name, owner_id, notion_page_id, notion_db_id,
		        budget_jpy, budget_suica, start_date, end_date, invite_code, created_at
		 FROM trips WHERE id = ?`, id)
	return scanTrip(row)
}

// GetTripByInviteCode returns a trip by its invite code.
func (d *DB) GetTripByInviteCode(ctx context.Context, code string) (*model.Trip, error) {
	row := d.QueryRowContext(ctx,
		`SELECT id, name, owner_id, notion_page_id, notion_db_id,
		        budget_jpy, budget_suica, start_date, end_date, invite_code, created_at
		 FROM trips WHERE invite_code = ?`, code)
	return scanTrip(row)
}

// ListTripsByUser returns all trips the user is a member of, newest first.
func (d *DB) ListTripsByUser(ctx context.Context, userID string) ([]model.Trip, error) {
	rows, err := d.QueryContext(ctx,
		`SELECT t.id, t.name, t.owner_id, t.notion_page_id, t.notion_db_id,
		        t.budget_jpy, t.budget_suica, t.start_date, t.end_date, t.invite_code, t.created_at
		 FROM trips t
		 JOIN members m ON t.id = m.trip_id
		 WHERE m.user_id = ?
		 ORDER BY t.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trips []model.Trip
	for rows.Next() {
		t, err := scanTrip(rows)
		if err != nil {
			return nil, err
		}
		trips = append(trips, *t)
	}
	return trips, rows.Err()
}

// UpdateTrip applies non-nil fields from req to the trip row.
func (d *DB) UpdateTrip(ctx context.Context, id string, req model.UpdateTripRequest) error {
	if req.Name != nil {
		if _, err := d.ExecContext(ctx, `UPDATE trips SET name=? WHERE id=?`, *req.Name, id); err != nil {
			return err
		}
	}
	if req.StartDate != nil {
		if _, err := d.ExecContext(ctx, `UPDATE trips SET start_date=? WHERE id=?`, *req.StartDate, id); err != nil {
			return err
		}
	}
	if req.EndDate != nil {
		if _, err := d.ExecContext(ctx, `UPDATE trips SET end_date=? WHERE id=?`, *req.EndDate, id); err != nil {
			return err
		}
	}
	if req.BudgetJPY != nil {
		if _, err := d.ExecContext(ctx, `UPDATE trips SET budget_jpy=? WHERE id=?`, *req.BudgetJPY, id); err != nil {
			return err
		}
	}
	if req.BudgetSuica != nil {
		if _, err := d.ExecContext(ctx, `UPDATE trips SET budget_suica=? WHERE id=?`, *req.BudgetSuica, id); err != nil {
			return err
		}
	}
	return nil
}

// DeleteTrip removes a trip and cascades to members.
func (d *DB) DeleteTrip(ctx context.Context, id string) error {
	res, err := d.ExecContext(ctx, `DELETE FROM trips WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func scanTrip(s scanner) (*model.Trip, error) {
	var t model.Trip
	err := s.Scan(
		&t.ID, &t.Name, &t.OwnerID, &t.NotionPageID, &t.NotionDbID,
		&t.BudgetJPY, &t.BudgetSuica, &t.StartDate, &t.EndDate, &t.InviteCode, &t.CreatedAt,
	)
	if isNoRows(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan trip: %w", err)
	}
	return &t, nil
}
