package interfaces

import (
	"context"
	"paypal-proxy/internal/domain/entities"
)

// OrderRepository defines the interface for order data access
type OrderRepository interface {
	// GetByID retrieves an order by its ID
	GetByID(ctx context.Context, id string) (*entities.Order, error)
	
	// Create creates a new order
	Create(ctx context.Context, order *entities.Order) (*entities.Order, error)
	
	// Update updates an existing order
	Update(ctx context.Context, id string, order *entities.Order) error
	
	// UpdateStatus updates only the order status
	UpdateStatus(ctx context.Context, id string, status entities.OrderStatus) error
	
	// UpdatePaymentInfo updates payment-related information
	UpdatePaymentInfo(ctx context.Context, id string, payment *entities.Payment) error
}

// PaymentRepository defines the interface for payment data access
type PaymentRepository interface {
	// Create creates a new payment record
	Create(ctx context.Context, payment *entities.Payment) error
	
	// GetByOrderID retrieves payments for a specific order
	GetByOrderID(ctx context.Context, orderID string) ([]*entities.Payment, error)
	
	// GetByPaymentID retrieves a payment by its payment provider ID
	GetByPaymentID(ctx context.Context, paymentID string) (*entities.Payment, error)
	
	// UpdateStatus updates the payment status
	UpdateStatus(ctx context.Context, paymentID string, status entities.PaymentStatus) error
}

// WooCommerceRepository defines the interface for WooCommerce API operations
type WooCommerceRepository interface {
	// MagicSpore operations (source store)
	GetMagicOrder(ctx context.Context, orderID string) (*entities.Order, error)
	UpdateMagicOrder(ctx context.Context, orderID string, order *entities.Order) error
	UpdateMagicOrderStatus(ctx context.Context, orderID string, status entities.OrderStatus) error
	UpdateMagicOrderPayment(ctx context.Context, orderID string, payment *entities.Payment) error
	
	// OITAM operations (payment processor store)
	CreateOITAMOrder(ctx context.Context, order *entities.Order) (*entities.Order, error)
	GetOITAMOrder(ctx context.Context, orderID string) (*entities.Order, error)
	UpdateOITAMOrder(ctx context.Context, orderID string, order *entities.Order) error
}