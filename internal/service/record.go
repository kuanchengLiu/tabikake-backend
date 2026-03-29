package service

import (
	"context"
	"fmt"

	"github.com/yourname/tabikake/internal/claude"
	"github.com/yourname/tabikake/internal/model"
	"github.com/yourname/tabikake/internal/notion"
	"github.com/yourname/tabikake/internal/store"
)

// RecordService handles expense record CRUD via Notion.
type RecordService struct {
	db     *store.DB
	notion *notion.Client
	claude *claude.Client
}

// NewRecordService creates a new RecordService.
func NewRecordService(db *store.DB, notionClient *notion.Client, claudeClient *claude.Client) *RecordService {
	return &RecordService{db: db, notion: notionClient, claude: claudeClient}
}

// ParseReceipt runs Claude Vision OCR on the provided image and returns structured data.
// The image must be base64-encoded.
func (s *RecordService) ParseReceipt(ctx context.Context, imageBase64, mediaType string) (*model.ParseReceiptResult, error) {
	return s.claude.ParseReceipt(ctx, imageBase64, mediaType)
}

// ListRecords fetches all records for a trip from Notion, enriching each with PaidByUser.
func (s *RecordService) ListRecords(ctx context.Context, tripID string) ([]model.Record, error) {
	trip, err := s.db.GetTrip(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("get trip: %w", err)
	}

	records, err := s.notion.ListRecords(ctx, trip.NotionDbID)
	if err != nil {
		return nil, err
	}

	// Collect all user IDs from records.
	idSet := make(map[string]struct{}, len(records))
	for _, r := range records {
		if r.PaidByUserID != "" {
			idSet[r.PaidByUserID] = struct{}{}
		}
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	users, _ := s.db.GetUsersByIDs(ctx, ids)
	for i := range records {
		if u, ok := users[records[i].PaidByUserID]; ok {
			u := u
			records[i].PaidByUser = &u
		}
	}

	return records, nil
}

// CreateRecord writes a confirmed expense record to the trip's Notion database.
func (s *RecordService) CreateRecord(ctx context.Context, req model.CreateRecordRequest) (*model.Record, error) {
	trip, err := s.db.GetTrip(ctx, req.TripID)
	if err != nil {
		return nil, fmt.Errorf("get trip: %w", err)
	}

	// Validate paid_by_user_id belongs to the trip.
	if req.PaidByUserID != "" {
		ok, err := s.db.IsMember(ctx, req.TripID, req.PaidByUserID)
		if err != nil {
			return nil, fmt.Errorf("check member: %w", err)
		}
		if !ok {
			return nil, &validationError{"paid_by_user_id is not a member of this trip"}
		}
	}

	return s.notion.CreateRecord(ctx, trip.NotionDbID, req)
}

// UpdateRecord updates fields on an existing Notion record page.
func (s *RecordService) UpdateRecord(ctx context.Context, pageID string, req model.UpdateRecordRequest) (*model.Record, error) {
	return s.notion.UpdateRecord(ctx, pageID, req)
}

// DeleteRecord archives a Notion record page.
func (s *RecordService) DeleteRecord(ctx context.Context, pageID string) error {
	return s.notion.DeleteRecord(ctx, pageID)
}

type validationError struct{ msg string }

func (e *validationError) Error() string { return e.msg }

// IsValidationError reports whether err is a field-validation error.
func IsValidationError(err error) bool {
	_, ok := err.(*validationError)
	return ok
}
