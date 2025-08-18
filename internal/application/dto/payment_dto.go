package dto

import "time"

// PaymentRedirectRequest represents the request to redirect to PayPal
type PaymentRedirectRequest struct {
	OrderID string `json:"order_id" validate:"required"`
	Domain  string `json:"domain"`
}

// PaymentRedirectResponse represents the response with redirect URL
type PaymentRedirectResponse struct {
	RedirectURL  string `json:"redirect_url"`
	OrderID      string `json:"order_id"`
	ProxyOrderID string `json:"proxy_order_id"`
	Status       string `json:"status"`
	Message      string `json:"message,omitempty"`
}

// PaymentReturnRequest represents the request from PayPal return
type PaymentReturnRequest struct {
	OrderID       string `json:"order_id"`
	OITAMOrderID  string `json:"oitam_order_id"`
	Status        string `json:"status"`
	PaymentID     string `json:"payment_id"`
	PayerID       string `json:"payer_id"`
	TransactionID string `json:"transaction_id"`
}

// PaymentReturnResponse represents the response after PayPal return
type PaymentReturnResponse struct {
	RedirectURL string `json:"redirect_url"`
	Status      string `json:"status"`
	Message     string `json:"message,omitempty"`
}

// PaymentCancelRequest represents the request from PayPal cancel
type PaymentCancelRequest struct {
	OrderID      string `json:"order_id"`
	OITAMOrderID string `json:"oitam_order_id"`
}

// PaymentCancelResponse represents the response after PayPal cancel
type PaymentCancelResponse struct {
	RedirectURL string `json:"redirect_url"`
	Status      string `json:"status"`
	Message     string `json:"message,omitempty"`
}

// WebhookRequest represents a webhook request
type WebhookRequest struct {
	EventType string                 `json:"event_type"`
	Resource  map[string]interface{} `json:"resource"`
	ID        string                 `json:"id"`
	CreateTime time.Time             `json:"create_time"`
}

// WebhookResponse represents a webhook response
type WebhookResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// OrderStatusRequest represents a request to get order status
type OrderStatusRequest struct {
	OrderID string `json:"order_id" validate:"required"`
}

// OrderStatusResponse represents the order status response
type OrderStatusResponse struct {
	OrderID       string    `json:"order_id"`
	Status        string    `json:"status"`
	Total         float64   `json:"total"`
	Currency      string    `json:"currency"`
	PaymentMethod string    `json:"payment_method"`
	DateCreated   time.Time `json:"date_created"`
	DatePaid      *time.Time `json:"date_paid"`
	TransactionID string    `json:"transaction_id,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Uptime    string    `json:"uptime"`
}