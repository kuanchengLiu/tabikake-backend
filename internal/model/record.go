package model

// RecordItem represents a single line item on a receipt.
type RecordItem struct {
	NameZH string  `json:"name_zh"`
	Price  float64 `json:"price"`
}

// Record represents an expense record stored in a trip's Notion database.
type Record struct {
	ID           string       `json:"id"`
	StoreNameZH  string       `json:"store_name_zh"`
	StoreNameJP  string       `json:"store_name_jp,omitempty"`
	AmountJPY    float64      `json:"amount_jpy"`
	TaxJPY       float64      `json:"tax_jpy"`
	Date         string       `json:"date"`
	Category     string       `json:"category"`
	Payment      string       `json:"payment"`
	PaidByUserID string       `json:"paid_by_user_id"`
	PaidByUser   *User        `json:"paid_by_user,omitempty"`
	SplitWith    []string     `json:"split_with"`
	Items        []RecordItem `json:"items"`
}

// CreateRecordRequest is the request body for POST /records.
type CreateRecordRequest struct {
	TripID       string       `json:"trip_id"`
	StoreNameZH  string       `json:"store_name_zh"`
	StoreNameJP  string       `json:"store_name_jp"`
	AmountJPY    float64      `json:"amount_jpy"`
	TaxJPY       float64      `json:"tax_jpy"`
	Date         string       `json:"date"`
	Category     string       `json:"category"`
	Payment      string       `json:"payment"`
	PaidByUserID string       `json:"paid_by_user_id"`
	SplitWith    []string     `json:"split_with"`
	Items        []RecordItem `json:"items"`
}

// UpdateRecordRequest is the request body for PATCH /records/:id (all fields optional).
type UpdateRecordRequest struct {
	StoreNameZH  *string      `json:"store_name_zh"`
	StoreNameJP  *string      `json:"store_name_jp"`
	AmountJPY    *float64     `json:"amount_jpy"`
	TaxJPY       *float64     `json:"tax_jpy"`
	Date         *string      `json:"date"`
	Category     *string      `json:"category"`
	Payment      *string      `json:"payment"`
	PaidByUserID *string      `json:"paid_by_user_id"`
	SplitWith    []string     `json:"split_with"`
	Items        []RecordItem `json:"items"`
}

// ParseReceiptResult is the structured JSON returned by Claude Vision OCR.
type ParseReceiptResult struct {
	StoreNameZH   string       `json:"store_name_zh"`
	StoreNameJP   string       `json:"store_name_jp"`
	AmountJPY     float64      `json:"amount_jpy"`
	TaxJPY        float64      `json:"tax_jpy"`
	PaymentMethod string       `json:"payment_method"`
	Category      string       `json:"category"`
	Items         []RecordItem `json:"items"`
	Date          string       `json:"date"`
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
