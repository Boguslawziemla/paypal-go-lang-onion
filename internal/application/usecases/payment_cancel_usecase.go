package usecases

import (
	"context"
	"fmt"
	"paypal-proxy/internal/application/dto"
	"paypal-proxy/internal/domain/entities"
	"paypal-proxy/internal/domain/interfaces"
	"paypal-proxy/internal/domain/services"
)

// PaymentCancelUseCase handles the payment cancel use case
type PaymentCancelUseCase struct {
	wooCommerceRepo interfaces.WooCommerceRepository
	paymentService  *services.PaymentDomainService
	orderService    *services.OrderDomainService
	logger          interfaces.Logger
	config          interfaces.ConfigService
}

// NewPaymentCancelUseCase creates a new payment cancel use case
func NewPaymentCancelUseCase(
	wooCommerceRepo interfaces.WooCommerceRepository,
	paymentService *services.PaymentDomainService,
	orderService *services.OrderDomainService,
	logger interfaces.Logger,
	config interfaces.ConfigService,
) *PaymentCancelUseCase {
	return &PaymentCancelUseCase{
		wooCommerceRepo: wooCommerceRepo,
		paymentService:  paymentService,
		orderService:    orderService,
		logger:          logger,
		config:          config,
	}
}

// Execute executes the payment cancel use case
func (uc *PaymentCancelUseCase) Execute(ctx context.Context, request *dto.PaymentCancelRequest) (*dto.PaymentCancelResponse, error) {
	uc.logger.Info("Processing payment cancellation", map[string]interface{}{
		"order_id":       request.OrderID,
		"oitam_order_id": request.OITAMOrderID,
	})

	returnURLs := uc.config.GetReturnURLs()

	// Update order status if order ID is provided
	if request.OrderID != "" {
		// Create a cancelled payment record
		cancelledPayment := uc.paymentService.CreatePaymentRecord(
			ctx,
			request.OrderID,
			"", // No payment ID for cancelled payments
			"", // No payer ID for cancelled payments
			entities.Money{}, // No amount for cancelled payments
			entities.PaymentStatusCancelled,
		)

		// Store the cancelled payment record (commented out as we don't have a payment repository yet)
		// TODO: Implement payment repository for storing payment records
		// if err := uc.paymentRepo.Create(ctx, cancelledPayment); err != nil {
		//     uc.logger.Error("Failed to store cancelled payment record", err, map[string]interface{}{
		//         "order_id": request.OrderID,
		//     })
		//     // Continue even if storage fails
		// }

		// Update original order status to cancelled
		err := uc.wooCommerceRepo.UpdateMagicOrderStatus(ctx, request.OrderID, entities.StatusCancelled)
		if err != nil {
			uc.logger.Error("Failed to update order status to cancelled", err, map[string]interface{}{
				"order_id": request.OrderID,
			})
			// Don't fail the entire operation if order update fails
		} else {
			uc.logger.Info("Order status updated to cancelled", map[string]interface{}{
				"order_id": request.OrderID,
			})
		}
	}

	// Build cancel redirect URL
	cancelURL := fmt.Sprintf("%s?order=%s&payment=cancelled", returnURLs.Cancel, request.OrderID)

	uc.logger.Info("Payment cancellation processed", map[string]interface{}{
		"order_id":     request.OrderID,
		"redirect_url": cancelURL,
	})

	return &dto.PaymentCancelResponse{
		RedirectURL: cancelURL,
		Status:      "cancelled",
		Message:     "Payment cancelled by customer",
	}, nil
}