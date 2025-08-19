package entities

import (
	"fmt"
	"strconv"
	"strings"
)

// Money represents a monetary value with currency
type Money struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// NewMoney creates a new Money instance
func NewMoney(amount float64, currency string) Money {
	return Money{
		Amount:   amount,
		Currency: strings.ToUpper(currency),
	}
}

// String returns string representation of money
func (m Money) String() string {
	return fmt.Sprintf("%.2f %s", m.Amount, m.Currency)
}

// IsZero checks if the money amount is zero
func (m Money) IsZero() bool {
	return m.Amount == 0
}

// IsPositive checks if the money amount is positive
func (m Money) IsPositive() bool {
	return m.Amount > 0
}

// Add adds two Money values (must have same currency)
func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("cannot add different currencies: %s and %s", m.Currency, other.Currency)
	}
	return Money{
		Amount:   m.Amount + other.Amount,
		Currency: m.Currency,
	}, nil
}

// Multiply multiplies money by a factor
func (m Money) Multiply(factor float64) Money {
	return Money{
		Amount:   m.Amount * factor,
		Currency: m.Currency,
	}
}

// ToWooCommerceFormat converts to WooCommerce API format
func (m Money) ToWooCommerceFormat() string {
	return fmt.Sprintf("%.2f", m.Amount)
}

// FromWooCommerceFormat parses WooCommerce API format
func FromWooCommerceFormat(amount, currency string) (Money, error) {
	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return Money{}, fmt.Errorf("invalid amount format: %s", amount)
	}
	return NewMoney(amountFloat, currency), nil
}