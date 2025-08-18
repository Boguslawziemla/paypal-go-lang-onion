package entities

import "time"

// Payment represents a payment transaction
type Payment struct {
	ID            string
	OrderID       string
	PaymentID     string
	PayerID       string
	Amount        Money
	Status        PaymentStatus
	Method        PaymentMethod
	TransactionID string
	ProcessedAt   time.Time
	Metadata      map[string]interface{}
}

// PaymentStatus represents the status of a payment
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusCancelled PaymentStatus = "cancelled"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)

// PaymentMethod represents the payment method
type PaymentMethod string

const (
	PaymentMethodPayPal PaymentMethod = "paypal"
	PaymentMethodStripe PaymentMethod = "stripe"
	PaymentMethodCard   PaymentMethod = "card"
)

// PaymentRequest represents a request to process a payment
type PaymentRequest struct {
	OrderID     string
	Amount      Money
	Method      PaymentMethod
	ReturnURL   string
	CancelURL   string
	Description string
	Metadata    map[string]interface{}
}

// PaymentResponse represents the response from a payment request
type PaymentResponse struct {
	PaymentID   string
	RedirectURL string
	Status      PaymentStatus
	Message     string
}

// IsSuccessful checks if the payment is successful
func (p *Payment) IsSuccessful() bool {
	return p.Status == PaymentStatusCompleted
}

// IsFinal checks if the payment is in a final state (no more processing needed)
func (p *Payment) IsFinal() bool {
	finalStatuses := []PaymentStatus{
		PaymentStatusCompleted,
		PaymentStatusFailed,
		PaymentStatusCancelled,
		PaymentStatusRefunded,
	}
	
	for _, status := range finalStatuses {
		if p.Status == status {
			return true
		}
	}
	return false
}