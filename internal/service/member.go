package service

import (
	"context"
	"fmt"

	"github.com/yourname/tabikake/internal/model"
	"github.com/yourname/tabikake/internal/store"
)

// MemberService handles trip membership queries.
type MemberService struct {
	db *store.DB
}

// NewMemberService creates a new MemberService.
func NewMemberService(db *store.DB) *MemberService {
	return &MemberService{db: db}
}

// ListMembers returns all members of a trip enriched with user profiles.
func (s *MemberService) ListMembers(ctx context.Context, tripID string) ([]model.MemberWithUser, error) {
	members, err := s.db.ListMembers(ctx, tripID)
	if err != nil {
		return nil, err
	}

	// Collect all user IDs.
	ids := make([]string, 0, len(members))
	for _, m := range members {
		ids = append(ids, m.UserID)
	}

	users, err := s.db.GetUsersByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("fetch users: %w", err)
	}

	result := make([]model.MemberWithUser, 0, len(members))
	for _, m := range members {
		mwu := model.MemberWithUser{Member: m}
		if u, ok := users[m.UserID]; ok {
			mwu.User = u
		}
		result = append(result, mwu)
	}
	return result, nil
}

// RemoveMember removes a user from a trip.
// Returns an error if the target user is the owner.
func (s *MemberService) RemoveMember(ctx context.Context, tripID, targetUserID string) error {
	isOwner, err := s.db.IsOwner(ctx, tripID, targetUserID)
	if err != nil {
		return err
	}
	if isOwner {
		return fmt.Errorf("cannot remove the trip owner")
	}
	return s.db.DeleteMember(ctx, tripID, targetUserID)
}

// IsMember returns whether userID is a member of tripID.
func (s *MemberService) IsMember(ctx context.Context, tripID, userID string) (bool, error) {
	return s.db.IsMember(ctx, tripID, userID)
}

// IsOwner returns whether userID is the owner of tripID.
func (s *MemberService) IsOwner(ctx context.Context, tripID, userID string) (bool, error) {
	return s.db.IsOwner(ctx, tripID, userID)
}
