// Package valueobject contains value objects that represent concepts without identity.
// Value objects are immutable and compared by their attributes rather than identity.
// They encapsulate validation logic and ensure data integrity.
//
// Value Objects follow these principles:
//   - Immutability: Once created, they cannot be changed.
//   - Equality: Two value objects are equal if all their attributes are equal.
//   - Self-validation: They validate their own data upon creation.
//   - Side-effect free: Methods returns new instances rather than modifying state
package valueobject

import (
	"errors"
	"fmt"
)

// Currency represents a monetary currency using ISO 4217 codes.
type Currency string

// Supported currencies in the system.
const (
	CurrencyUSD Currency = "USD" // US Dollar
	CurrencyEUR Currency = "EUR" // Euro
	CurrencyGBP Currency = "GBP" // British Pound
	CurrencyAED Currency = "AED" // UAE Dirham
	CurrencyCOP Currency = "COP" // Colombian Peso
)

// Money errors define domain-specific error conditions.
var (
	ErrInvalidCurrency  = errors.New("invalid currency code")
	ErrCurrencyMismatch = errors.New("currency mismatch in operation")
	ErrNegativeAmount   = errors.New("money amount cannot be negative")
	ErrDivisionByZero   = errors.New("cannot divide by zero")
)

// Money represents a monetary value with currency.
// It stores amounts in the smallest unit (cents) to avoid floating-point issues.
//
// Example usage:
//
//	price := valueobject.NewMoney(1999, valueobject.CurrencyUSD) // $19.99
//	total := price.Multiply(3) // $59.97
type Money struct {
	// Amount in smallest currency unit (e.g., cents for USD)
	Amount int64 `json:"amount"`

	// Currency using ISO 4217 code
	Currency Currency `json:"currency"`
}

// NewMoney creates a new Money value object.
//
// Parameters:
//   - amount: Amount in smallest unit (e.g., cents)
//   - currency: ISO 4217 currency code
//
// Returns:
//   - Money: the created Money value object
func NewMoney(amount int64, currency Currency) Money {
	return Money{
		Amount:   amount,
		Currency: currency,
	}
}

// NewMoneyFromFloat creates a new Money from a decimal amount.
//
// Parameters:
//   - amount: Decimal amount (e.g., 19.99)
//   - currency: ISO 4217 currency code
//
// Returns:
//   - Money: the created Money value object
func NewMoneyFromFloat(amount float64, currency Currency) Money {
	// Convert to cents (assuming 2 decimal places)
	cents := int64(amount * 100)
	return NewMoney(cents, currency)
}

// Zero returns a zero-value Money in the specified currency.
//
// Parameters:
//   - currency: ISO 4217 currency code
//
// Returns:
//   - Money: Zero Money in the specified currency
func Zero(currency Currency) Money {
	return NewMoney(0, currency)
}

// Add adds two Money values and returns a new Money.
// Both values must have the same currency.
//
// Parameters:
//   - other: the Money to add
//
// Returns:
//   - Money: the sum of the two Money values
//
// Note: Panics if currencies do not match. Use AddSafe for error handling.
func (m Money) Add(other Money) Money {
	if m.Currency != other.Currency && !m.IsZero() && !other.IsZero() {
		panic(ErrCurrencyMismatch)
	}
	currency := m.Currency
	if m.IsZero() {
		currency = other.Currency
	}
	return NewMoney(m.Amount+other.Amount, currency)
}

// AddSafe adds two Money values with error handling.
// Both values must have the same currency.
//
// Parameters:
//   - other: the Money to add
//
// Returns:
//   - Money: the sum of the two Money values
//   - error: ErrCurrencyMismatch if currencies do not match
func (m Money) AddSafe(other Money) (Money, error) {
	if m.Currency != other.Currency && !m.IsZero() && !other.IsZero() {
		return Money{}, ErrCurrencyMismatch
	}
	currency := m.Currency
	if m.IsZero() {
		currency = other.Currency
	}
	return NewMoney(m.Amount+other.Amount, currency), nil
}

// Subtract subtracts another Money from this Money and returns a new Money.
// Both values must have the same currency.
//
// Parameters:
//   - other: the Money to subtract
//
// Returns:
//   - Money: Difference of both values
func (m Money) Subtract(other Money) Money {
	if m.Currency != other.Currency && !m.IsZero() && !other.IsZero() {
		panic(ErrCurrencyMismatch)
	}
	return NewMoney(m.Amount-other.Amount, m.Currency)
}

// Multiply multiplies the Money amount by a factor and returns a new Money.
//
// Parameters:
//   - factor: the multiplication factor
//
// Returns:
//   - Money: the multiplied Money value
func (m Money) Multiply(factor int) Money {
	return NewMoney(m.Amount*int64(factor), m.Currency)
}

// MultiplyFloat multiplies the Money amount by a float factor.
// Useful for applying percentage discounts.
//
// Parameters:
//   - factor: the multiplication factor as float
//
// Returns:
//   - Money: the multiplied Money value (rounded to nearest cent)
func (m Money) MultiplyFloat(factor float64) Money {
	return NewMoney(int64(float64(m.Amount)*factor), m.Currency)
}

// Divide divides the Money amount by a divisor and returns a new Money.
//
// Parameters:
//   - divisor: the division factor (must not be zero)
//
// Returns:
//   - Money: the divided Money value
//   - error: ErrDivisionByZero if divisor is zero
func (m Money) Divide(divisor int) (Money, error) {
	if divisor == 0 {
		return Money{}, ErrDivisionByZero
	}
	return NewMoney(m.Amount/int64(divisor), m.Currency), nil
}

// Percentage calculates the given percentage of the Money amount.
//
// Parameters:
//   - percent: the percentage to calculate (e.g., 15.0 for 15%)
//
// Returns:
//   - Money: the calculated percentage
func (m Money) Percentage(percent float64) Money {
	return m.MultiplyFloat(percent / 100)
}

// IsZero checks if the Money amount is zero.
//
// Returns:
//   - bool: true if amount is zero
func (m Money) IsZero() bool {
	return m.Amount == 0
}

// IsPositive checks if the Money amount is positive.
//
// Returns:
//   - bool: true if amount is greater than zero
func (m Money) IsPositive() bool {
	return m.Amount > 0
}

// IsNegative checks if the Money amount is negative.
//
// Returns:
//   - bool: true if amount is less than zero
func (m Money) IsNegative() bool {
	return m.Amount < 0
}

// Equals checks if two Money values are equal in amount and currency.
//
// Parameters:
//   - other: the Money to compare
//
// Returns:
//   - bool: true if both Money values are equal
func (m Money) Equals(other Money) bool {
	return m.Amount == other.Amount && m.Currency == other.Currency
}

// GreaterThan checks if this Money is greater than another Money.
//
// Parameters:
//   - other: the Money to compare (must have same currency)
//
// Returns:
//   - bool: true if this Money is greater
func (m Money) GreaterThan(other Money) bool {
	if m.Currency != other.Currency {
		panic(ErrCurrencyMismatch)
	}
	return m.Amount > other.Amount
}

// LessThan checks if this Money is less than another Money.
//
// Parameters:
//   - other: the Money to compare (must have same currency)
//
// Returns:
//   - bool: true if this Money is less
func (m Money) LessThan(other Money) bool {
	if m.Currency != other.Currency {
		panic(ErrCurrencyMismatch)
	}
	return m.Amount < other.Amount
}

// Negate returns a new Money with the negated amount.
//
// Returns:
//   - Money: new Money with negated amount
func (m Money) Negate() Money {
	return NewMoney(-m.Amount, m.Currency)
}

// Abs returns a new Money with the absolute amount.
//
// Returns:
//   - Money: new Money with absolute amount
func (m Money) Abs() Money {
	if m.Amount < 0 {
		return m.Negate()
	}
	return m
}

// ToFloat converts the Money amount to a float64 representation.
//
// Returns:
//   - float64: Decimal representation (e.g., 19.99)
func (m Money) ToFloat() float64 {
	return float64(m.Amount) / 100.0
}

// String returns a formatted string representation of the Money.
//
// Returns:
//   - string: Formatted string (e.g., "USD 19.99")
func (m Money) String() string {
	return fmt.Sprintf("%s %.2f", m.Currency, m.ToFloat())
}

// Format returns the money formatted with its currency symbol.
//
// Returns:
//   - string: Formatted string with currency symbol (e.g., "$19.99")
func (m Money) Format() string {
	symbol := currencySymbol(m.Currency)
	return fmt.Sprintf("%s%.2f", symbol, m.ToFloat())
}

// currencySymbol returns the symbol for a given currency.
func currencySymbol(c Currency) string {
	symbols := map[Currency]string{
		CurrencyUSD: "$",
		CurrencyEUR: "€",
		CurrencyGBP: "£",
		CurrencyAED: "د.إ",
		CurrencyCOP: "$",
	}

	if symbol, ok := symbols[c]; ok {
		return symbol
	}
	return string(c) + " "
}

// Split divides the Money amount into n nearly equal parts.
// Handles reminder by distributing extra cents to the first parts.
//
// Parameters:
//   - n: number of parts to split into
//
// Returns:
//   - []Money: slice of n Money values that sum to the original
//   - error: ErrDivisionByZero if n is zero
func (m Money) Split(n int) ([]Money, error) {
	if n == 0 {
		return nil, ErrDivisionByZero
	}

	baseAmount := m.Amount / int64(n)
	remainder := m.Amount % int64(n)

	parts := make([]Money, n)
	for i := 0; i < n; i++ {
		amount := baseAmount
		if int64(i) < remainder {
			amount++ // Distribute remainder to first parts
		}
		parts[i] = NewMoney(amount, m.Currency)

	}

	return parts, nil
}

// Allocate distributes money according to the given ratios.
// Useful for splitting payments or applying discounts.
//
// Parameters:
//   - ratios: slice of ratios (e.g., [1, 2, 1] splits 25%, 50%, 25%)
//
// Returns:
//   - []Money: Allocated amounts
//   - error: ErrDivisionByZero if total ratio is zero
func (m Money) Allocate(ratios []int) ([]Money, error) {
	totalRatio := 0
	for _, r := range ratios {
		totalRatio += r
	}

	if totalRatio == 0 {
		return nil, ErrDivisionByZero
	}

	allocated := make([]Money, len(ratios))
	remainder := m.Amount

	for i, ratio := range ratios {
		share := m.Amount * int64(ratio) / int64(totalRatio)
		allocated[i] = NewMoney(share, m.Currency)
		remainder -= share
	}

	// Distribute remainder to first allocations
	if remainder > 0 {
		allocated[0] = NewMoney(allocated[0].Amount+remainder, m.Currency)
	}

	return allocated, nil
}
