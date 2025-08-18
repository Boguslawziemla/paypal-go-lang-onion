package interfaces

import (
	"context"
	"paypal-proxy/internal/domain/entities"
)

// PaymentGateway defines the interface for payment processing
type PaymentGateway interface {
	// CreatePayment initiates a payment process
	CreatePayment(ctx context.Context, request *entities.PaymentRequest) (*entities.PaymentResponse, error)
	
	// ProcessPayment processes a payment
	ProcessPayment(ctx context.Context, paymentID string, payerID string) (*entities.Payment, error)
	
	// GetPaymentStatus retrieves the current status of a payment
	GetPaymentStatus(ctx context.Context, paymentID string) (*entities.Payment, error)
	
	// CancelPayment cancels a payment
	CancelPayment(ctx context.Context, paymentID string) error
	
	// RefundPayment refunds a payment
	RefundPayment(ctx context.Context, paymentID string, amount entities.Money) error
}

// URLBuilder defines the interface for building URLs
type URLBuilder interface {
	// BuildCheckoutURL builds a checkout URL
	BuildCheckoutURL(order *entities.Order, returnURL, cancelURL string) string
	
	// BuildReturnURL builds a return URL after payment
	BuildReturnURL(baseURL string, orderID string, paymentID string, status string) string
	
	// BuildCancelURL builds a cancel URL
	BuildCancelURL(baseURL string, orderID string) string
}

// NotificationService defines the interface for sending notifications
type NotificationService interface {
	// SendOrderUpdate sends an order update notification
	SendOrderUpdate(ctx context.Context, order *entities.Order) error
	
	// SendPaymentSuccess sends a payment success notification
	SendPaymentSuccess(ctx context.Context, order *entities.Order, payment *entities.Payment) error
	
	// SendPaymentFailure sends a payment failure notification
	SendPaymentFailure(ctx context.Context, order *entities.Order, reason string) error
}

// Logger defines the interface for logging
type Logger interface {
	// Debug logs a debug message
	Debug(message string, fields map[string]interface{})
	
	// Info logs an info message
	Info(message string, fields map[string]interface{})
	
	// Warn logs a warning message
	Warn(message string, fields map[string]interface{})
	
	// Error logs an error message
	Error(message string, err error, fields map[string]interface{})
}

// ConfigService defines the interface for configuration
type ConfigService interface {
	// GetMagicSporeConfig returns MagicSpore configuration
	GetMagicSporeConfig() MagicSporeConfig
	
	// GetOITAMConfig returns OITAM configuration
	GetOITAMConfig() OITAMConfig
	
	// GetReturnURLs returns return URL configuration
	GetReturnURLs() ReturnURLsConfig
	
	// GetServerConfig returns server configuration
	GetServerConfig() ServerConfig
}

// Configuration types
type MagicSporeConfig struct {
	APIURL        string
	ConsumerKey   string
	ConsumerSecret string
}

type OITAMConfig struct {
	APIURL        string
	ConsumerKey   string
	ConsumerSecret string
	CheckoutURL   string
}

type ReturnURLsConfig struct {
	Success string
	Cancel  string
	Error   string
}

type ServerConfig struct {
	Environment string
	Port        string
	LogLevel    string
	APITimeout  int
	RateLimit   int
}