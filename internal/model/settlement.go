package model

// SettlementItem represents a single transfer needed to settle debts.
type SettlementItem struct {
	From      User  `json:"from"`
	To        User  `json:"to"`
	AmountJPY int64 `json:"amount_jpy"`
}

// UserBalance holds a user's net balance for settlement calculation.
type UserBalance struct {
	User    User
	Balance int64 // positive = others owe them; negative = they owe others
}

// SettlementResult is returned by GET /settlement/:trip_id.
type SettlementResult struct {
	TotalJPY    int64            `json:"total_jpy"`
	Settlements []SettlementItem `json:"settlements"`
}

// ExportSettlementResponse is returned by POST /settlement/:trip_id/export.
type ExportSettlementResponse struct {
	NotionPageURL string `json:"notion_page_url"`
}
