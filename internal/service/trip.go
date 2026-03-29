package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/yourname/tabikake/internal/db"
	"github.com/yourname/tabikake/internal/model"
	"github.com/yourname/tabikake/internal/notion"
)

// TripService handles trip creation and retrieval.
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

// IsMember returns true if memberID belongs to the given trip.
func (s *TripService) IsMember(ctx context.Context, tripID, memberID string) (bool, error) {
	return s.db.IsMember(ctx, tripID, memberID)
}

// GetJoinInfo returns public trip info for the given invite code (no auth required).
func (s *TripService) GetJoinInfo(ctx context.Context, inviteCode string) (*model.JoinInfoResponse, error) {
	trip, err := s.db.GetTripByInviteCode(ctx, inviteCode)
	if err != nil {
		return nil, err
	}

	owner, err := s.db.GetTripOwner(ctx, trip.ID)
	if err != nil {
		return nil, err
	}

	count, err := s.db.CountMembers(ctx, trip.ID)
	if err != nil {
		return nil, err
	}

	return &model.JoinInfoResponse{
		TripName:    trip.Name,
		MemberCount: count,
		OwnerName:   owner.Name,
	}, nil
}

// CreateTrip:
//  1. Creates a Notion page under NOTION_ROOT_PAGE_ID
//  2. Creates a Records database under that page
//  3. Saves the trip metadata to SQLite (with auto-generated invite code)
//  4. Creates an owner Member record in SQLite
func (s *TripService) CreateTrip(ctx context.Context, req model.CreateTripRequest) (*model.CreateTripResponse, error) {
	pageID, err := s.notion.CreateTripPage(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("create notion page: %w", err)
	}

	dbID, err := s.notion.CreateRecordsDatabase(ctx, pageID)
	if err != nil {
		return nil, fmt.Errorf("create records database: %w", err)
	}

	tripID, err := newID()
	if err != nil {
		return nil, err
	}

	inviteCode, err := newInviteCode()
	if err != nil {
		return nil, err
	}

	trip := model.Trip{
		ID:           tripID,
		Name:         req.Name,
		StartDate:    req.StartDate,
		EndDate:      req.EndDate,
		NotionPageID: pageID,
		NotionDbID:   dbID,
		InviteCode:   inviteCode,
	}

	if err := s.db.InsertTrip(ctx, trip); err != nil {
		return nil, fmt.Errorf("save trip: %w", err)
	}

	ownerID, err := newID()
	if err != nil {
		return nil, err
	}

	color := req.OwnerAvatarColor
	if color == "" {
		color = "#F59E0B"
	}

	owner := model.Member{
		ID:          ownerID,
		TripID:      tripID,
		Name:        req.OwnerName,
		AvatarColor: color,
		IsOwner:     true,
	}

	if err := s.db.InsertMember(ctx, owner); err != nil {
		return nil, fmt.Errorf("save owner member: %w", err)
	}

	return &model.CreateTripResponse{Trip: trip, Owner: owner}, nil
}

func newID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

const inviteChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func newInviteCode() (string, error) {
	code := make([]byte, 6)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(inviteChars))))
		if err != nil {
			return "", fmt.Errorf("generate invite code: %w", err)
		}
		code[i] = inviteChars[n.Int64()]
	}
	return string(code), nil
}
