package services

import (
	"context"
	"errors"
	"fmt"
	"paypal-proxy/internal/domain/entities"
	"paypal-proxy/internal/domain/interfaces"
	"time"
)

// PaymentDomainService implements core business logic for payments
type PaymentDomainService struct {
	logger interfaces.Logger
}

// NewPaymentDomainService creates a new payment domain service
func NewPaymentDomainService(logger interfaces.Logger) *PaymentDomainService {
	return &PaymentDomainService{
		logger: logger,
	}
}

// CreatePaymentRequest creates a payment request for an order
func (s *PaymentDomainService) CreatePaymentRequest(ctx context.Context, order *entities.Order, returnURL, cancelURL string) (*entities.PaymentRequest, error) {
	if order == nil {
		return nil, errors.New("order cannot be nil")
	}
	
	if returnURL == "" {
		return nil, errors.New("return URL is required")
	}
	
	if cancelURL == "" {
		return nil, errors.New("cancel URL is required")
	}
	
	request := &entities.PaymentRequest{
		OrderID:     string(rune(order.ID)),
		Amount:      order.Total,
		Method:      entities.PaymentMethodPayPal,
		ReturnURL:   returnURL,
		CancelURL:   cancelURL,
		Description: s.generatePaymentDescription(order),
		Metadata: map[string]interface{}{
			"order_number":    order.Number,
			"original_order_id": order.ID,
			"currency":        order.Currency,
			"items_count":     len(order.LineItems),
		},
	}
	
	s.logger.Info("Payment request created", map[string]interface{}{
		"order_id":    order.ID,
		"amount":      request.Amount.Amount,
		"currency":    request.Amount.Currency,
		"method":      request.Method,
	})
	
	return request, nil
}

// ProcessPaymentResult processes the result of a payment operation
func (s *PaymentDomainService) ProcessPaymentResult(ctx context.Context, payment *entities.Payment, order *entities.Order) error {
	if payment == nil {
		return errors.New("payment cannot be nil")
	}
	
	if order == nil {
		return errors.New("order cannot be nil")
	}
	
	// Validate payment matches order
	if payment.OrderID != string(rune(order.ID)) {
		return errors.New("payment order ID does not match order")
	}
	
	// Validate payment amount matches order total
	if payment.Amount.Amount != order.Total.Amount {
		s.logger.Warn("Payment amount mismatch", map[string]interface{}{
			"order_id":        order.ID,
			"payment_amount":  payment.Amount.Amount,
			"order_amount":    order.Total.Amount,
		})
	}
	
	// Validate currency
	if payment.Amount.Currency != order.Currency {
		return errors.New("payment currency does not match order currency")
	}
	
	s.logger.Info("Payment result processed", map[string]interface{}{
		"payment_id":   payment.ID,
		"order_id":     order.ID,
		"status":       payment.Status,
		"amount":       payment.Amount.Amount,
	})
	
	return nil
}

// ValidateWebhookPayment validates a payment from webhook data
func (s *PaymentDomainService) ValidateWebhookPayment(ctx context.Context, payment *entities.Payment, expectedOrderID string) error {
	if payment == nil {
		return errors.New("payment cannot be nil")
	}
	
	if payment.PaymentID == "" {
		return errors.New("payment ID is required")
	}
	
	if payment.OrderID != expectedOrderID {
		return errors.New("payment order ID does not match expected order ID")
	}
	
	if payment.Amount.Amount <= 0 {
		return errors.New("payment amount must be greater than zero")
	}
	
	if payment.Amount.Currency == "" {
		return errors.New("payment currency is required")
	}
	
	if !payment.IsFinal() {
		return errors.New("payment is not in a final state")
	}
	
	s.logger.Info("Webhook payment validated", map[string]interface{}{
		"payment_id": payment.PaymentID,
		"order_id":   payment.OrderID,
		"status":     payment.Status,
		"amount":     payment.Amount.Amount,
	})
	
	return nil
}

// CreatePaymentRecord creates a payment record from payment provider data
func (s *PaymentDomainService) CreatePaymentRecord(ctx context.Context, orderID, paymentID, payerID string, amount entities.Money, status entities.PaymentStatus) *entities.Payment {
	payment := &entities.Payment{
		ID:            generatePaymentID(),
		OrderID:       orderID,
		PaymentID:     paymentID,
		PayerID:       payerID,
		Amount:        amount,
		Status:        status,
		Method:        entities.PaymentMethodPayPal,
		TransactionID: paymentID, // Use PayPal payment ID as transaction ID
		ProcessedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"payment_provider": "paypal",
			"processed_at":     time.Now().Format(time.RFC3339),
		},
	}
	
	if payerID != "" {
		payment.Metadata["payer_id"] = payerID
	}
	
	s.logger.Info("Payment record created", map[string]interface{}{
		"payment_id":      payment.ID,
		"order_id":        payment.OrderID,
		"paypal_payment_id": payment.PaymentID,
		"status":          payment.Status,
	})
	
	return payment
}

// generatePaymentDescription creates a description for the payment
func (s *PaymentDomainService) generatePaymentDescription(order *entities.Order) string {
	itemCount := len(order.LineItems)
	if itemCount == 1 {
		return "Payment for 1 item"
	}
	return fmt.Sprintf("Payment for %d items", itemCount)
}

// generatePaymentID generates a unique payment ID
func generatePaymentID() string {
	// In a real implementation, you might use UUID or another unique ID generator
	return fmt.Sprintf("pay_%d", time.Now().UnixNano())
}