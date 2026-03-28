package service

import (
	"context"
	"fmt"

	appdb "github.com/yourname/tabikake/internal/db"
	"github.com/yourname/tabikake/internal/model"
)

// MemberService handles member management for trips.
type MemberService struct {
	db *appdb.DB
}

// NewMemberService creates a new MemberService.
func NewMemberService(database *appdb.DB) *MemberService {
	return &MemberService{db: database}
}

// ListMembers returns all members for a trip.
func (s *MemberService) ListMembers(ctx context.Context, tripID string) ([]model.Member, error) {
	// Verify trip exists.
	if _, err := s.db.GetTrip(ctx, tripID); err != nil {
		return nil, err
	}
	members, err := s.db.ListMembers(ctx, tripID)
	if err != nil {
		return nil, err
	}
	if members == nil {
		members = []model.Member{}
	}
	return members, nil
}

// AddMember adds a new member to a trip (called by the trip owner).
func (s *MemberService) AddMember(ctx context.Context, tripID string, req model.CreateMemberRequest) (*model.Member, error) {
	if _, err := s.db.GetTrip(ctx, tripID); err != nil {
		return nil, err
	}

	id, err := newID()
	if err != nil {
		return nil, err
	}

	color := req.AvatarColor
	if color == "" {
		color = "#F59E0B"
	}

	m := model.Member{
		ID:          id,
		TripID:      tripID,
		Name:        req.Name,
		AvatarColor: color,
		IsOwner:     false,
	}

	if err := s.db.InsertMember(ctx, m); err != nil {
		return nil, fmt.Errorf("insert member: %w", err)
	}
	return &m, nil
}

// JoinTrip creates a new member record via invite code and returns the trip + member.
func (s *MemberService) JoinTrip(ctx context.Context, req model.JoinTripRequest) (*model.JoinTripResponse, error) {
	trip, err := s.db.GetTripByInviteCode(ctx, req.InviteCode)
	if err != nil {
		return nil, err
	}

	id, err := newID()
	if err != nil {
		return nil, err
	}

	color := req.AvatarColor
	if color == "" {
		color = "#F59E0B"
	}

	m := model.Member{
		ID:          id,
		TripID:      trip.ID,
		Name:        req.Name,
		AvatarColor: color,
		IsOwner:     false,
	}

	if err := s.db.InsertMember(ctx, m); err != nil {
		return nil, fmt.Errorf("insert member: %w", err)
	}

	return &model.JoinTripResponse{Trip: *trip, Member: m}, nil
}

// DeleteMember removes a member from a trip (owner only — authorization checked in handler).
func (s *MemberService) DeleteMember(ctx context.Context, tripID, memberID string) error {
	m, err := s.db.GetMember(ctx, memberID)
	if err != nil {
		return err
	}
	if m.TripID != tripID {
		return appdb.ErrNotFound
	}
	if m.IsOwner {
		return fmt.Errorf("cannot remove trip owner")
	}
	return s.db.DeleteMember(ctx, memberID)
}
