package entities

import (
	"time"
)

// PaymentStatus represents the status of a payment
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusCreated   PaymentStatus = "created"
	PaymentStatusApproved  PaymentStatus = "approved"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusCancelled PaymentStatus = "cancelled"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)

// PaymentMethod represents the payment method
type PaymentMethod string

const (
	PaymentMethodPayPal     PaymentMethod = "paypal"
	PaymentMethodCreditCard PaymentMethod = "credit_card"
	PaymentMethodBankTransfer PaymentMethod = "bank_transfer"
)

// Payment represents a payment transaction
type Payment struct {
	ID              string        `json:"id"`
	OrderID         string        `json:"order_id"`
	PaymentID       string        `json:"payment_id"`       // PayPal payment ID
	PayerID         string        `json:"payer_id"`         // PayPal payer ID
	TransactionID   string        `json:"transaction_id"`   // PayPal transaction ID
	Status          PaymentStatus `json:"status"`
	Method          PaymentMethod `json:"method"`
	Amount          Money         `json:"amount"`
	Currency        string        `json:"currency"`
	Description     string        `json:"description"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	CompletedAt     *time.Time    `json:"completed_at"`
	
	// PayPal specific fields
	PayPalDetails   *PayPalDetails `json:"paypal_details,omitempty"`
	
	// URLs
	ApprovalURL     string `json:"approval_url,omitempty"`
	ReturnURL       string `json:"return_url,omitempty"`
	CancelURL       string `json:"cancel_url,omitempty"`
	
	// Additional info
	FailureReason   string `json:"failure_reason,omitempty"`
	RefundAmount    *Money `json:"refund_amount,omitempty"`
	
	// Metadata
	MetaData        []MetaData `json:"meta_data,omitempty"`
}

// PayPalDetails contains PayPal-specific payment details
type PayPalDetails struct {
	PaymentID      string                 `json:"payment_id"`
	State          string                 `json:"state"`
	Intent         string                 `json:"intent"`
	Payer          *PayPalPayer          `json:"payer,omitempty"`
	Transactions   []*PayPalTransaction  `json:"transactions,omitempty"`
	Links          []*PayPalLink         `json:"links,omitempty"`
	CreateTime     time.Time             `json:"create_time"`
	UpdateTime     time.Time             `json:"update_time"`
}

// PayPalPayer contains payer information from PayPal
type PayPalPayer struct {
	PaymentMethod string              `json:"payment_method"`
	Status        string              `json:"status"`
	PayerInfo     *PayPalPayerInfo   `json:"payer_info,omitempty"`
}

// PayPalPayerInfo contains detailed payer information
type PayPalPayerInfo struct {
	Email       string `json:"email"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	PayerID     string `json:"payer_id"`
	CountryCode string `json:"country_code"`
}

// PayPalTransaction contains transaction details
type PayPalTransaction struct {
	Amount      *PayPalAmount `json:"amount"`
	Description string        `json:"description"`
	InvoiceNumber string      `json:"invoice_number"`
}

// PayPalAmount contains amount details
type PayPalAmount struct {
	Total    string `json:"total"`
	Currency string `json:"currency"`
}

// PayPalLink contains PayPal navigation links
type PayPalLink struct {
	Href   string `json:"href"`
	Rel    string `json:"rel"`
	Method string `json:"method"`
}

// PaymentRequest represents a payment creation request
type PaymentRequest struct {
	OrderID     string    `json:"order_id"`
	Amount      Money     `json:"amount"`
	Currency    string    `json:"currency"`
	Description string    `json:"description"`
	ReturnURL   string    `json:"return_url"`
	CancelURL   string    `json:"cancel_url"`
	Method      PaymentMethod `json:"method"`
}

// PaymentResponse represents a payment creation response
type PaymentResponse struct {
	PaymentID   string `json:"payment_id"`
	Status      PaymentStatus `json:"status"`
	ApprovalURL string `json:"approval_url"`
	CreatedAt   time.Time `json:"created_at"`
}

// IsCompleted checks if the payment is completed
func (p *Payment) IsCompleted() bool {
	return p.Status == PaymentStatusCompleted
}

// IsPending checks if the payment is pending
func (p *Payment) IsPending() bool {
	return p.Status == PaymentStatusPending || p.Status == PaymentStatusCreated
}

// IsApproved checks if the payment is approved by the payer
func (p *Payment) IsApproved() bool {
	return p.Status == PaymentStatusApproved
}

// IsFailed checks if the payment has failed
func (p *Payment) IsFailed() bool {
	return p.Status == PaymentStatusFailed || p.Status == PaymentStatusCancelled
}

// CanBeProcessed checks if the payment can be processed
func (p *Payment) CanBeProcessed() bool {
	return p.Status == PaymentStatusApproved && p.PayerID != ""
}

// MarkAsCompleted marks the payment as completed
func (p *Payment) MarkAsCompleted(transactionID string) {
	p.Status = PaymentStatusCompleted
	p.TransactionID = transactionID
	now := time.Now()
	p.CompletedAt = &now
	p.UpdatedAt = now
}

// MarkAsFailed marks the payment as failed
func (p *Payment) MarkAsFailed(reason string) {
	p.Status = PaymentStatusFailed
	p.FailureReason = reason
	p.UpdatedAt = time.Now()
}

// GetApprovalURL returns the PayPal approval URL
func (p *Payment) GetApprovalURL() string {
	if p.ApprovalURL != "" {
		return p.ApprovalURL
	}
	
	// Try to extract from PayPal links
	if p.PayPalDetails != nil {
		for _, link := range p.PayPalDetails.Links {
			if link.Rel == "approval_url" {
				return link.Href
			}
		}
	}
	
	return ""
}

// Validate validates the payment entity
func (p *Payment) Validate() error {
	if p.OrderID == "" {
		return NewValidationError("order_id", "Order ID is required")
	}
	
	if p.Amount.Amount <= 0 {
		return NewValidationError("amount", "Payment amount must be positive")
	}
	
	if p.Currency == "" {
		return NewValidationError("currency", "Currency is required")
	}
	
	return nil
}

// NewPayment creates a new payment instance
func NewPayment(orderID string, amount Money, method PaymentMethod) *Payment {
	return &Payment{
		OrderID:   orderID,
		Amount:    amount,
		Currency:  amount.Currency,
		Method:    method,
		Status:    PaymentStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Message
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}