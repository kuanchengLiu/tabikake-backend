package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/yourname/tabikake/internal/db"
	"github.com/yourname/tabikake/internal/model"
	"github.com/yourname/tabikake/internal/notion"
)

// TripService handles trip creation and retrieval.
// Trips are persisted in SQLite; their Notion pages/databases are created on demand.
type TripService struct {
	db     *db.DB
	notion *notion.Client
}

// NewTripService creates a new TripService.
func NewTripService(database *db.DB, notionClient *notion.Client) *TripService {
	return &TripService{db: database, notion: notionClient}
}

// ListTrips returns all trips from SQLite.
func (s *TripService) ListTrips(ctx context.Context) ([]model.Trip, error) {
	return s.db.ListTrips(ctx)
}

// GetTrip returns a single trip by ID from SQLite.
func (s *TripService) GetTrip(ctx context.Context, id string) (*model.Trip, error) {
	return s.db.GetTrip(ctx, id)
}

// CreateTrip:
//  1. Creates a Notion page under NOTION_ROOT_PAGE_ID
//  2. Creates a Records database under that page
//  3. Saves the trip metadata to SQLite
func (s *TripService) CreateTrip(ctx context.Context, req model.CreateTripRequest) (*model.Trip, error) {
	pageID, err := s.notion.CreateTripPage(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("create notion page: %w", err)
	}

	dbID, err := s.notion.CreateRecordsDatabase(ctx, pageID)
	if err != nil {
		return nil, fmt.Errorf("create records database: %w", err)
	}

	id, err := newID()
	if err != nil {
		return nil, err
	}

	trip := model.Trip{
		ID:           id,
		Name:         req.Name,
		StartDate:    req.StartDate,
		EndDate:      req.EndDate,
		NotionPageID: pageID,
		NotionDbID:   dbID,
	}

	if err := s.db.InsertTrip(ctx, trip); err != nil {
		return nil, fmt.Errorf("save trip: %w", err)
	}

	return &trip, nil
}

func newID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
