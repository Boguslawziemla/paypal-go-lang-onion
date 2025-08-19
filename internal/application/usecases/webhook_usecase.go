package usecases

import (
	"context"
	"fmt"
	"paypal-proxy/internal/application/dto"
	"paypal-proxy/internal/domain/entities"
	"paypal-proxy/internal/domain/interfaces"
	"paypal-proxy/internal/domain/services"
)

// WebhookUseCase handles webhook processing use case
type WebhookUseCase struct {
	wooCommerceRepo interfaces.WooCommerceRepository
	paymentService  *services.PaymentDomainService
	orderService    *services.OrderDomainService
	logger          interfaces.Logger
	config          interfaces.ConfigService
}

// NewWebhookUseCase creates a new webhook use case
func NewWebhookUseCase(
	wooCommerceRepo interfaces.WooCommerceRepository,
	paymentService *services.PaymentDomainService,
	orderService *services.OrderDomainService,
	logger interfaces.Logger,
	config interfaces.ConfigService,
) *WebhookUseCase {
	return &WebhookUseCase{
		wooCommerceRepo: wooCommerceRepo,
		paymentService:  paymentService,
		orderService:    orderService,
		logger:          logger,
		config:          config,
	}
}

// Execute executes the webhook processing use case
func (uc *WebhookUseCase) Execute(ctx context.Context, request *dto.WebhookRequest) (*dto.WebhookResponse, error) {
	uc.logger.Info("Processing webhook", map[string]interface{}{
		"event_type": request.EventType,
		"webhook_id": request.ID,
	})

	switch request.EventType {
	case "PAYMENT.CAPTURE.COMPLETED":
		return uc.handlePaymentCaptureCompleted(ctx, request)
	case "PAYMENT.CAPTURE.DENIED":
		return uc.handlePaymentCaptureDenied(ctx, request)
	case "PAYMENT.CAPTURE.REFUNDED":
		return uc.handlePaymentCaptureRefunded(ctx, request)
	default:
		uc.logger.Info("Unhandled webhook event type", map[string]interface{}{
			"event_type": request.EventType,
			"webhook_id": request.ID,
		})
		return &dto.WebhookResponse{
			Status:  "ignored",
			Message: fmt.Sprintf("Event type %s not handled", request.EventType),
		}, nil
	}
}

// handlePaymentCaptureCompleted handles payment capture completed webhook
func (uc *WebhookUseCase) handlePaymentCaptureCompleted(ctx context.Context, request *dto.WebhookRequest) (*dto.WebhookResponse, error) {
	// Extract payment information from webhook
	paymentID, ok := request.Resource["id"].(string)
	if !ok {
		return nil, fmt.Errorf("payment ID not found in webhook")
	}

	// Try to get order ID from custom_id or invoice_id
	var orderID string
	if customID, exists := request.Resource["custom_id"].(string); exists {
		orderID = customID
	} else if invoiceID, exists := request.Resource["invoice_id"].(string); exists {
		orderID = invoiceID
	}

	if orderID == "" {
		return nil, fmt.Errorf("order ID not found in webhook")
	}

	uc.logger.Info("Processing payment capture completed", map[string]interface{}{
		"payment_id": paymentID,
		"order_id":   orderID,
	})

	// Extract amount information
	amount := entities.Money{}
	if amountData, exists := request.Resource["amount"].(map[string]interface{}); exists {
		if value, ok := amountData["value"].(string); ok {
			// Convert string to float64
			if parsedValue, err := parseFloat(value); err == nil {
				amount.Amount = parsedValue
			}
		}
		if currency, ok := amountData["currency_code"].(string); ok {
			amount.Currency = currency
		}
	}

	// Create payment record
	payment := uc.paymentService.CreatePaymentRecord(
		ctx,
		orderID,
		paymentID,
		"", // No payer ID in webhook typically
		amount,
		entities.PaymentStatusCompleted,
	)

	// Store payment record (commented out as we don't have a payment repository yet)
	// TODO: Implement payment repository for storing payment records
	// if err := uc.paymentRepo.Create(ctx, payment); err != nil {
	//     uc.logger.Error("Failed to store webhook payment record", err, map[string]interface{}{
	//         "payment_id": payment.ID,
	//         "order_id":   orderID,
	//     })
	//     // Continue even if storage fails
	// }

	// Update original order status to completed
	if err := uc.wooCommerceRepo.UpdateMagicOrderPayment(ctx, orderID, payment); err != nil {
		uc.logger.Error("Failed to update order from webhook", err, map[string]interface{}{
			"order_id":   orderID,
			"payment_id": paymentID,
		})
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	uc.logger.Info("Payment capture completed processed successfully", map[string]interface{}{
		"payment_id": paymentID,
		"order_id":   orderID,
		"amount":     amount.Amount,
	})

	return &dto.WebhookResponse{
		Status:  "processed",
		Message: "Payment capture completed processed successfully",
	}, nil
}

// handlePaymentCaptureDenied handles payment capture denied webhook
func (uc *WebhookUseCase) handlePaymentCaptureDenied(ctx context.Context, request *dto.WebhookRequest) (*dto.WebhookResponse, error) {
	// Extract payment information
	paymentID, _ := request.Resource["id"].(string)
	orderID := uc.extractOrderIDFromResource(request.Resource)

	uc.logger.Info("Processing payment capture denied", map[string]interface{}{
		"payment_id": paymentID,
		"order_id":   orderID,
	})

	if orderID != "" {
		// Update order status to failed
		if err := uc.wooCommerceRepo.UpdateMagicOrderStatus(ctx, orderID, entities.StatusFailed); err != nil {
			uc.logger.Error("Failed to update order status to failed", err, map[string]interface{}{
				"order_id": orderID,
			})
		}
	}

	return &dto.WebhookResponse{
		Status:  "processed",
		Message: "Payment capture denied processed",
	}, nil
}

// handlePaymentCaptureRefunded handles payment capture refunded webhook
func (uc *WebhookUseCase) handlePaymentCaptureRefunded(ctx context.Context, request *dto.WebhookRequest) (*dto.WebhookResponse, error) {
	// Extract payment information
	paymentID, _ := request.Resource["id"].(string)
	orderID := uc.extractOrderIDFromResource(request.Resource)

	uc.logger.Info("Processing payment capture refunded", map[string]interface{}{
		"payment_id": paymentID,
		"order_id":   orderID,
	})

	if orderID != "" {
		// Update order status to refunded
		if err := uc.wooCommerceRepo.UpdateMagicOrderStatus(ctx, orderID, entities.StatusRefunded); err != nil {
			uc.logger.Error("Failed to update order status to refunded", err, map[string]interface{}{
				"order_id": orderID,
			})
		}
	}

	return &dto.WebhookResponse{
		Status:  "processed",
		Message: "Payment capture refunded processed",
	}, nil
}

// extractOrderIDFromResource extracts order ID from webhook resource
func (uc *WebhookUseCase) extractOrderIDFromResource(resource map[string]interface{}) string {
	// Try custom_id first
	if customID, exists := resource["custom_id"].(string); exists {
		return customID
	}
	
	// Try invoice_id
	if invoiceID, exists := resource["invoice_id"].(string); exists {
		return invoiceID
	}
	
	return ""
}

// parseFloat safely parses a string to float64
func parseFloat(s string) (float64, error) {
	// Simple implementation - in production use strconv.ParseFloat
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}