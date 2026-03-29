package model

// Member represents a trip participant, linking a User to a Trip in SQLite.
type Member struct {
	ID       string `json:"id"`
	TripID   string `json:"trip_id"`
	UserID   string `json:"user_id"`
	IsOwner  bool   `json:"is_owner"`
	JoinedAt string `json:"joined_at"`
}

// MemberWithUser combines a Member with its associated User profile.
type MemberWithUser struct {
	Member
	User User `json:"user"`
}

// JoinTripRequest is the request body for POST /trips/join.
type JoinTripRequest struct {
	InviteCode string `json:"invite_code"`
}

// JoinTripResponse is returned after joining a trip.
type JoinTripResponse struct {
	Trip   Trip   `json:"trip"`
	Member Member `json:"member"`
}
