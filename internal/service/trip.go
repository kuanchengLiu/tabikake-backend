package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/google/uuid"

	"github.com/yourname/tabikake/internal/model"
	"github.com/yourname/tabikake/internal/notion"
	"github.com/yourname/tabikake/internal/store"
)

// TripService handles trip CRUD and membership.
type TripService struct {
	db     *store.DB
	notion *notion.Client
}

// NewTripService creates a new TripService.
func NewTripService(db *store.DB, notionClient *notion.Client) *TripService {
	return &TripService{db: db, notion: notionClient}
}

// ListTrips returns all trips the user is a member of.
func (s *TripService) ListTrips(ctx context.Context, userID string) ([]model.Trip, error) {
	trips, err := s.db.ListTripsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if trips == nil {
		trips = []model.Trip{}
	}
	return trips, nil
}

// GetTrip returns a single trip by ID (caller must verify membership).
func (s *TripService) GetTrip(ctx context.Context, id string) (*model.Trip, error) {
	return s.db.GetTrip(ctx, id)
}

// CreateTrip creates a Notion page + DB, persists to SQLite, and adds the user as owner.
func (s *TripService) CreateTrip(ctx context.Context, userID string, req model.CreateTripRequest) (*model.Trip, error) {
	pageID, err := s.notion.CreateTripPage(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("create notion page: %w", err)
	}

	dbID, err := s.notion.CreateRecordsDatabase(ctx, pageID)
	if err != nil {
		return nil, fmt.Errorf("create records database: %w", err)
	}

	inviteCode, err := newInviteCode()
	if err != nil {
		return nil, err
	}

	trip := model.Trip{
		ID:           uuid.NewString(),
		Name:         req.Name,
		OwnerID:      userID,
		NotionPageID: pageID,
		NotionDbID:   dbID,
		BudgetJPY:    req.BudgetJPY,
		BudgetSuica:  req.BudgetSuica,
		StartDate:    req.StartDate,
		EndDate:      req.EndDate,
		InviteCode:   inviteCode,
	}

	if err := s.db.InsertTrip(ctx, trip); err != nil {
		return nil, fmt.Errorf("save trip: %w", err)
	}

	// Auto-add creator as owner member.
	owner := model.Member{
		ID:      uuid.NewString(),
		TripID:  trip.ID,
		UserID:  userID,
		IsOwner: true,
	}
	if err := s.db.InsertMember(ctx, owner); err != nil {
		return nil, fmt.Errorf("add owner member: %w", err)
	}

	return &trip, nil
}

// UpdateTrip updates mutable trip fields (owner only — enforced in handler).
func (s *TripService) UpdateTrip(ctx context.Context, id string, req model.UpdateTripRequest) (*model.Trip, error) {
	if err := s.db.UpdateTrip(ctx, id, req); err != nil {
		return nil, err
	}
	return s.db.GetTrip(ctx, id)
}

// DeleteTrip removes a trip from SQLite (cascades to members).
func (s *TripService) DeleteTrip(ctx context.Context, id string) error {
	return s.db.DeleteTrip(ctx, id)
}

// GetJoinInfo returns public preview info for a trip by invite code.
func (s *TripService) GetJoinInfo(ctx context.Context, inviteCode string) (*model.JoinInfoResponse, error) {
	trip, err := s.db.GetTripByInviteCode(ctx, inviteCode)
	if err != nil {
		return nil, err
	}
	ownerName, err := s.db.GetTripOwnerName(ctx, trip.ID)
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
		OwnerName:   ownerName,
	}, nil
}

// JoinTrip adds userID to the trip identified by invite code.
func (s *TripService) JoinTrip(ctx context.Context, userID string, req model.JoinTripRequest) (*model.JoinTripResponse, error) {
	trip, err := s.db.GetTripByInviteCode(ctx, req.InviteCode)
	if err != nil {
		return nil, err
	}

	member := model.Member{
		ID:      uuid.NewString(),
		TripID:  trip.ID,
		UserID:  userID,
		IsOwner: false,
	}
	if err := s.db.InsertMember(ctx, member); err != nil {
		if err == store.ErrConflict {
			// Already a member — return existing trip info anyway.
			return &model.JoinTripResponse{Trip: *trip, Member: member}, nil
		}
		return nil, fmt.Errorf("join trip: %w", err)
	}

	return &model.JoinTripResponse{Trip: *trip, Member: member}, nil
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

// newHexID generates a random 8-byte hex string (kept for compatibility).
func newHexID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

var _ = newHexID // suppress unused warning
