package dto

// MoneyResponse represents a monetary value in API responses.
type MoneyResponse struct {
	// Amount in smallest currency unit
	Amount int64 `json:"amount"`

	// Currency code
	Currency string `json:"currency"`

	// Formatted is the display string
	Formatted string `json:"formatted"`
}
