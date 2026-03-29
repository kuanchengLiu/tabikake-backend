package model

// --- Auth ---

// NotionUser represents the authenticated user via Notion OAuth.
type NotionUser struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Email     string `json:"email"`
}

// JWTClaims holds the JWT payload stored in the token.
type JWTClaims struct {
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
}

// AuthResponse is the response returned after successful OAuth login.
type AuthResponse struct {
	Token string     `json:"token"`
	User  NotionUser `json:"user"`
}

// --- Trip ---

// Trip represents a travel trip stored in SQLite.
type Trip struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
	NotionPageID string `json:"notion_page_id"`
	NotionDbID   string `json:"notion_db_id"`
	InviteCode   string `json:"invite_code"`
	CreatedAt    string `json:"created_at"`
}

// CreateTripRequest is the request body for POST /trips.
type CreateTripRequest struct {
	Name             string `json:"name"`
	StartDate        string `json:"start_date"`
	EndDate          string `json:"end_date"`
	OwnerName        string `json:"owner_name"`
	OwnerAvatarColor string `json:"owner_avatar_color"`
}

// CreateTripResponse is returned after POST /trips, including the auto-created owner member.
type CreateTripResponse struct {
	Trip  Trip   `json:"trip"`
	Owner Member `json:"owner"`
}

// TripDetailResponse is returned by GET /trips/:id and includes membership status.
type TripDetailResponse struct {
	Trip
	IsMember bool `json:"is_member"`
}

// JoinInfoResponse is returned by GET /trips/join-info?code= (public endpoint).
type JoinInfoResponse struct {
	TripName    string `json:"trip_name"`
	MemberCount int    `json:"member_count"`
	OwnerName   string `json:"owner_name"`
}

// --- Member ---

// Member represents a trip participant stored in SQLite.
type Member struct {
	ID          string `json:"id"`
	TripID      string `json:"trip_id"`
	Name        string `json:"name"`
	AvatarColor string `json:"avatar_color"`
	IsOwner     bool   `json:"is_owner"`
	CreatedAt   string `json:"created_at"`
}

// CreateMemberRequest is the request body for POST /trips/:id/members.
type CreateMemberRequest struct {
	Name        string `json:"name"`
	AvatarColor string `json:"avatar_color"`
}

// JoinTripRequest is the request body for POST /trips/join.
type JoinTripRequest struct {
	InviteCode  string `json:"invite_code"`
	Name        string `json:"name"`
	AvatarColor string `json:"avatar_color"`
}

// JoinTripResponse is returned after joining a trip.
type JoinTripResponse struct {
	Trip   Trip   `json:"trip"`
	Member Member `json:"member"`
}

// --- Record ---

// RecordItem represents a single line item on a receipt.
type RecordItem struct {
	NameJP string  `json:"name_jp"`
	NameZH string  `json:"name_zh"`
	Price  float64 `json:"price"`
}

// Record represents an expense record stored in a trip's Notion Records database.
type Record struct {
	ID             string       `json:"id"`
	Store          string       `json:"store"`
	Date           string       `json:"date"`
	AmountJPY      float64      `json:"amount_jpy"`
	AmountTWD      float64      `json:"amount_twd"`
	TaxJPY         float64      `json:"tax_jpy"`
	Category       string       `json:"category"`
	Payment        string       `json:"payment"`
	PaidBy         string       `json:"paid_by"`
	PaidByName     string       `json:"paid_by_name"`
	PaidByMemberID string       `json:"paid_by_member_id,omitempty"`
	PaidByMember   *Member      `json:"paid_by_member,omitempty"`
	SplitWith      []string     `json:"split_with"`
	Items          []RecordItem `json:"items"`
}

// CreateRecordRequest is the request body for POST /records.
type CreateRecordRequest struct {
	TripID         string       `json:"trip_id"`
	Store          string       `json:"store"`
	Date           string       `json:"date"`
	AmountJPY      float64      `json:"amount_jpy"`
	AmountTWD      float64      `json:"amount_twd"`
	TaxJPY         float64      `json:"tax_jpy"`
	Category       string       `json:"category"`
	Payment        string       `json:"payment"`
	PaidBy         string       `json:"paid_by"`
	PaidByName     string       `json:"paid_by_name"`
	PaidByMemberID string       `json:"paid_by_member_id"`
	SplitWith      []string     `json:"split_with"`
	Items          []RecordItem `json:"items"`
	ImageBase64    string       `json:"image_base64,omitempty"`
}

// UpdateRecordRequest is the request body for PATCH /records/:id.
// All fields are optional; only non-nil fields are written to Notion.
type UpdateRecordRequest struct {
	Store          *string      `json:"store"`
	Date           *string      `json:"date"`
	AmountJPY      *float64     `json:"amount_jpy"`
	AmountTWD      *float64     `json:"amount_twd"`
	TaxJPY         *float64     `json:"tax_jpy"`
	Category       *string      `json:"category"`
	Payment        *string      `json:"payment"`
	PaidBy         *string      `json:"paid_by"`
	PaidByName     *string      `json:"paid_by_name"`
	PaidByMemberID *string      `json:"paid_by_member_id"`
	SplitWith      []string     `json:"split_with"`
	Items          []RecordItem `json:"items"`
}

// Category options
const (
	CategoryFood           = "餐飲"
	CategoryTransportation = "交通"
	CategoryShopping       = "購物"
	CategoryAccommodation  = "住宿"
	CategoryOther          = "其他"
)

// Payment method options
const (
	PaymentCash       = "現金"
	PaymentSuica      = "Suica"
	PaymentPayPay     = "PayPay"
	PaymentCreditCard = "信用卡"
)

// --- Parse (Claude Vision) ---

// ParseReceiptResult is the structured JSON returned by Claude after OCR.
type ParseReceiptResult struct {
	StoreNameJP   string       `json:"store_name_jp"`
	StoreNameZH   string       `json:"store_name_zh"`
	AmountJPY     float64      `json:"amount_jpy"`
	TaxJPY        float64      `json:"tax_jpy"`
	PaymentMethod string       `json:"payment_method"`
	Category      string       `json:"category"`
	Items         []RecordItem `json:"items"`
	Date          string       `json:"date"`
}

// --- Dashboard ---

// MemberBalance holds per-member spending summary for settlement.
type MemberBalance struct {
	UserID    string  `json:"user_id"`
	UserName  string  `json:"user_name"`
	TotalPaid float64 `json:"total_paid"`
	ShouldPay float64 `json:"should_pay"`
	Balance   float64 `json:"balance"`
}

// Settlement represents a single transfer needed to settle debts (dashboard/split export).
type Settlement struct {
	FromUserID   string  `json:"from_user_id"`
	FromUserName string  `json:"from_user_name"`
	ToUserID     string  `json:"to_user_id"`
	ToUserName   string  `json:"to_user_name"`
	Amount       float64 `json:"amount"`
}

// CategorySummary holds total spending per category.
type CategorySummary struct {
	Category  string  `json:"category"`
	AmountJPY float64 `json:"amount_jpy"`
}

// DashboardResponse is the response for GET /dashboard/:trip_id.
type DashboardResponse struct {
	TripID          string            `json:"trip_id"`
	TotalJPY        float64           `json:"total_jpy"`
	MemberBalances  []MemberBalance   `json:"member_balances"`
	CategorySummary []CategorySummary `json:"category_summary"`
	Settlements     []Settlement      `json:"settlements"`
}

// --- Member Settlement ---

// MemberSummary holds per-member spending summary using the Members table.
type MemberSummary struct {
	Member  Member `json:"member"`
	PaidJPY int64  `json:"paid_jpy"`
	OweJPY  int64  `json:"owe_jpy"`
	DiffJPY int64  `json:"diff_jpy"`
}

// MemberSettlement represents a single transfer between registered members.
type MemberSettlement struct {
	From      Member `json:"from"`
	To        Member `json:"to"`
	AmountJPY int64  `json:"amount_jpy"`
}

// SettlementResult is the response for GET /trips/:id/settlement.
type SettlementResult struct {
	TotalJPY    int64              `json:"total_jpy"`
	ByMember    []MemberSummary    `json:"by_member"`
	Settlements []MemberSettlement `json:"settlements"`
}

// --- Split Export ---

// ExportSettlementResponse is the response for POST /split/export/:trip_id.
type ExportSettlementResponse struct {
	NotionPageURL string `json:"notion_page_url"`
}

// --- Error ---

// ErrorResponse is the standard error response body.
type ErrorResponse struct {
	Error string `json:"error"`
}
