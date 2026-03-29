package service

import (
	"context"
	"fmt"

	appdb "github.com/yourname/tabikake/internal/db"
	"github.com/yourname/tabikake/internal/model"
	"github.com/yourname/tabikake/internal/notion"
)

// RecordService handles expense record CRUD.
type RecordService struct {
	db     *appdb.DB
	notion *notion.Client
}

// NewRecordService creates a new RecordService.
func NewRecordService(database *appdb.DB, notionClient *notion.Client) *RecordService {
	return &RecordService{db: database, notion: notionClient}
}

// ListRecords returns all records for the given trip, with PaidByMember populated.
func (s *RecordService) ListRecords(ctx context.Context, tripID string) ([]model.Record, error) {
	trip, err := s.db.GetTrip(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("get trip: %w", err)
	}

	records, err := s.notion.ListRecords(ctx, trip.NotionDbID)
	if err != nil {
		return nil, err
	}

	// Load all members once and index by ID.
	members, _ := s.db.ListMembers(ctx, tripID)
	memberMap := make(map[string]*model.Member, len(members))
	for i := range members {
		m := members[i]
		memberMap[m.ID] = &m
	}

	for i := range records {
		if mid := records[i].PaidByMemberID; mid != "" {
			if m, ok := memberMap[mid]; ok {
				records[i].PaidByMember = m
			}
		}
	}

	return records, nil
}

// CreateRecord writes a confirmed expense record to the trip's Notion database.
// Validates that paid_by_member_id (if provided) belongs to the trip.
func (s *RecordService) CreateRecord(ctx context.Context, req model.CreateRecordRequest) (*model.Record, error) {
	trip, err := s.db.GetTrip(ctx, req.TripID)
	if err != nil {
		return nil, fmt.Errorf("get trip: %w", err)
	}

	if req.PaidByMemberID != "" {
		ok, err := s.db.IsMember(ctx, req.TripID, req.PaidByMemberID)
		if err != nil {
			return nil, fmt.Errorf("check member: %w", err)
		}
		if !ok {
			return nil, errNotTripMember
		}

		// Auto-fill name from member record if not provided.
		if req.PaidByName == "" {
			if m, err := s.db.GetMember(ctx, req.PaidByMemberID); err == nil {
				req.PaidByName = m.Name
				if req.PaidBy == "" {
					req.PaidBy = m.ID
				}
			}
		}
	}

	return s.notion.CreateRecord(ctx, trip.NotionDbID, req)
}

// UpdateRecord updates an existing record page in Notion.
func (s *RecordService) UpdateRecord(ctx context.Context, pageID string, req model.UpdateRecordRequest) (*model.Record, error) {
	return s.notion.UpdateRecord(ctx, pageID, req)
}

// DeleteRecord archives the Notion page for the given record.
func (s *RecordService) DeleteRecord(ctx context.Context, pageID string) error {
	return s.notion.DeleteRecord(ctx, pageID)
}

// errNotTripMember is returned when paid_by_member_id does not belong to the trip.
var errNotTripMember = &memberValidationError{"paid_by_member_id is not a member of this trip"}

type memberValidationError struct{ msg string }

func (e *memberValidationError) Error() string { return e.msg }

// IsMemberValidationError reports whether err is a member validation error.
func IsMemberValidationError(err error) bool {
	_, ok := err.(*memberValidationError)
	return ok
}
