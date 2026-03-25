package service

import (
	"context"
	"fmt"

	"github.com/yourname/tabikake/internal/db"
	"github.com/yourname/tabikake/internal/model"
	"github.com/yourname/tabikake/internal/notion"
)

// RecordService handles expense record CRUD, routing writes to each trip's own Notion database.
type RecordService struct {
	db     *db.DB
	notion *notion.Client
}

// NewRecordService creates a new RecordService.
func NewRecordService(database *db.DB, notionClient *notion.Client) *RecordService {
	return &RecordService{db: database, notion: notionClient}
}

// ListRecords returns all records for the given trip from its Notion database.
func (s *RecordService) ListRecords(ctx context.Context, tripID string) ([]model.Record, error) {
	trip, err := s.db.GetTrip(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("get trip: %w", err)
	}
	return s.notion.ListRecords(ctx, trip.NotionDbID)
}

// CreateRecord writes a confirmed expense record to the trip's Notion database.
func (s *RecordService) CreateRecord(ctx context.Context, req model.CreateRecordRequest) (*model.Record, error) {
	trip, err := s.db.GetTrip(ctx, req.TripID)
	if err != nil {
		return nil, fmt.Errorf("get trip: %w", err)
	}
	return s.notion.CreateRecord(ctx, trip.NotionDbID, req)
}
