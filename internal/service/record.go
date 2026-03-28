package service

import (
	"context"
	"fmt"

	"github.com/yourname/tabikake/internal/db"
	"github.com/yourname/tabikake/internal/model"
	"github.com/yourname/tabikake/internal/notion"
)

// RecordService handles expense record CRUD.
type RecordService struct {
	db     *db.DB
	notion *notion.Client
}

// NewRecordService creates a new RecordService.
func NewRecordService(database *db.DB, notionClient *notion.Client) *RecordService {
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
func (s *RecordService) CreateRecord(ctx context.Context, req model.CreateRecordRequest) (*model.Record, error) {
	trip, err := s.db.GetTrip(ctx, req.TripID)
	if err != nil {
		return nil, fmt.Errorf("get trip: %w", err)
	}

	// If paid_by_member_id is provided, fill in name from member record.
	if req.PaidByMemberID != "" && req.PaidByName == "" {
		if m, err := s.db.GetMember(ctx, req.PaidByMemberID); err == nil {
			req.PaidByName = m.Name
			if req.PaidBy == "" {
				req.PaidBy = m.ID
			}
		}
	}

	return s.notion.CreateRecord(ctx, trip.NotionDbID, req)
}
