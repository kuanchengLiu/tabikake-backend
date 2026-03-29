package model

// ByMember holds per-user spending info for a trip.
type ByMember struct {
	User       User    `json:"user"`
	PaidJPY    int64   `json:"paid_jpy"`
	Percentage float64 `json:"percentage"`
}

// ByCategory holds total spending per category.
type ByCategory struct {
	Category  string `json:"category"`
	AmountJPY int64  `json:"amount_jpy"`
}

// ByDate holds total spending per date.
type ByDate struct {
	Date      string `json:"date"`
	AmountJPY int64  `json:"amount_jpy"`
}

// DashboardResponse is returned by GET /dashboard/:trip_id.
type DashboardResponse struct {
	TotalJPY   int64        `json:"total_jpy"`
	ByMember   []ByMember   `json:"by_member"`
	ByCategory []ByCategory `json:"by_category"`
	ByDate     []ByDate     `json:"by_date"`
	Records    []Record     `json:"records"`
}
