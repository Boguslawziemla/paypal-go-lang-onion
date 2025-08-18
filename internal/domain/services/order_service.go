package services

import (
	"context"
	"errors"
	"paypal-proxy/internal/domain/entities"
	"paypal-proxy/internal/domain/interfaces"
)

// OrderDomainService implements core business logic for orders
type OrderDomainService struct {
	logger interfaces.Logger
}

// NewOrderDomainService creates a new order domain service
func NewOrderDomainService(logger interfaces.Logger) *OrderDomainService {
	return &OrderDomainService{
		logger: logger,
	}
}

// ValidateOrderForPayment validates if an order can be processed for payment
func (s *OrderDomainService) ValidateOrderForPayment(ctx context.Context, order *entities.Order) error {
	if order == nil {
		return errors.New("order cannot be nil")
	}
	
	if order.ID == 0 {
		return errors.New("order ID is required")
	}
	
	if order.Total.Amount <= 0 {
		return errors.New("order total must be greater than zero")
	}
	
	if order.IsPaymentCompleted() {
		return errors.New("order payment is already completed")
	}
	
	if !order.CanBeProcessed() {
		return errors.New("order cannot be processed in current status")
	}
	
	if order.Currency == "" {
		return errors.New("order currency is required")
	}
	
	if len(order.LineItems) == 0 {
		return errors.New("order must have at least one line item")
	}
	
	s.logger.Info("Order validation successful", map[string]interface{}{
		"order_id":  order.ID,
		"order_total": order.Total.Amount,
		"currency":  order.Currency,
		"status":    order.Status,
	})
	
	return nil
}

// CreateAnonymousOrder creates an anonymized version of an order for proxy processing
func (s *OrderDomainService) CreateAnonymousOrder(ctx context.Context, originalOrder *entities.Order) (*entities.Order, error) {
	if originalOrder == nil {
		return nil, errors.New("original order cannot be nil")
	}
	
	// Validate the original order first
	if err := s.ValidateOrderForPayment(ctx, originalOrder); err != nil {
		return nil, err
	}
	
	anonymousOrder := originalOrder.ToAnonymousOrder()
	
	s.logger.Info("Anonymous order created", map[string]interface{}{
		"original_order_id": originalOrder.ID,
		"anonymous_order_number": anonymousOrder.Number,
		"line_items_count": len(anonymousOrder.LineItems),
	})
	
	return anonymousOrder, nil
}

// CalculateOrderTotals ensures order totals are consistent
func (s *OrderDomainService) CalculateOrderTotals(ctx context.Context, order *entities.Order) error {
	if order == nil {
		return errors.New("order cannot be nil")
	}
	
	var itemsTotal float64
	for _, item := range order.LineItems {
		itemsTotal += item.Total.Amount
	}
	
	var shippingTotal float64
	for _, shipping := range order.ShippingLines {
		shippingTotal += shipping.Total.Amount
	}
	
	var feesTotal float64
	for _, fee := range order.FeeLines {
		feesTotal += fee.Total.Amount
	}
	
	var taxTotal float64
	for _, tax := range order.TaxLines {
		taxTotal += tax.TaxTotal.Amount
	}
	
	var discountTotal float64
	for _, coupon := range order.CouponLines {
		discountTotal += coupon.Discount.Amount
	}
	
	calculatedTotal := itemsTotal + shippingTotal + feesTotal + taxTotal - discountTotal
	
	// Allow small floating point differences
	tolerance := 0.01
	if abs(calculatedTotal-order.Total.Amount) > tolerance {
		s.logger.Warn("Order total mismatch", map[string]interface{}{
			"order_id": order.ID,
			"calculated_total": calculatedTotal,
			"order_total": order.Total.Amount,
			"difference": abs(calculatedTotal - order.Total.Amount),
		})
	}
	
	return nil
}

// DetermineOrderStatus determines the appropriate order status based on payment
func (s *OrderDomainService) DetermineOrderStatus(ctx context.Context, order *entities.Order, payment *entities.Payment) entities.OrderStatus {
	if payment == nil {
		return entities.StatusPending
	}
	
	switch payment.Status {
	case entities.PaymentStatusCompleted:
		return entities.StatusProcessing
	case entities.PaymentStatusFailed:
		return entities.StatusFailed
	case entities.PaymentStatusCancelled:
		return entities.StatusCancelled
	case entities.PaymentStatusRefunded:
		return entities.StatusRefunded
	default:
		return entities.StatusPending
	}
}

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}