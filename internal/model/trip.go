package model

// Trip represents a travel trip stored in SQLite.
type Trip struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	OwnerID      string `json:"owner_id"`
	NotionPageID string `json:"notion_page_id"`
	NotionDbID   string `json:"notion_db_id"`
	BudgetJPY    int64  `json:"budget_jpy"`
	BudgetSuica  int64  `json:"budget_suica"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
	InviteCode   string `json:"invite_code"`
	CreatedAt    string `json:"created_at"`
}

// CreateTripRequest is the request body for POST /trips.
type CreateTripRequest struct {
	Name        string `json:"name"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	BudgetJPY   int64  `json:"budget_jpy"`
	BudgetSuica int64  `json:"budget_suica"`
}

// UpdateTripRequest is the request body for PATCH /trips/:id (all fields optional).
type UpdateTripRequest struct {
	Name        *string `json:"name"`
	StartDate   *string `json:"start_date"`
	EndDate     *string `json:"end_date"`
	BudgetJPY   *int64  `json:"budget_jpy"`
	BudgetSuica *int64  `json:"budget_suica"`
}

// JoinInfoResponse is returned by GET /trips/join-info?code= (public endpoint).
type JoinInfoResponse struct {
	TripName    string `json:"trip_name"`
	MemberCount int    `json:"member_count"`
	OwnerName   string `json:"owner_name"`
}
