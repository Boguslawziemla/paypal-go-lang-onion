package usecases

import (
	"context"
	"fmt"
	"paypal-proxy/internal/application/dto"
	"paypal-proxy/internal/domain/entities"
	"paypal-proxy/internal/domain/interfaces"
	"paypal-proxy/internal/domain/services"
)

// PaymentReturnUseCase handles the payment return use case
type PaymentReturnUseCase struct {
	wooCommerceRepo interfaces.WooCommerceRepository
	paymentRepo     interfaces.PaymentRepository
	paymentGateway  interfaces.PaymentGateway
	paymentService  *services.PaymentDomainService
	orderService    *services.OrderDomainService
	logger          interfaces.Logger
	config          interfaces.ConfigService
}

// NewPaymentReturnUseCase creates a new payment return use case
func NewPaymentReturnUseCase(
	wooCommerceRepo interfaces.WooCommerceRepository,
	paymentRepo interfaces.PaymentRepository,
	paymentGateway interfaces.PaymentGateway,
	paymentService *services.PaymentDomainService,
	orderService *services.OrderDomainService,
	logger interfaces.Logger,
	config interfaces.ConfigService,
) *PaymentReturnUseCase {
	return &PaymentReturnUseCase{
		wooCommerceRepo: wooCommerceRepo,
		paymentRepo:     paymentRepo,
		paymentGateway:  paymentGateway,
		paymentService:  paymentService,
		orderService:    orderService,
		logger:          logger,
		config:          config,
	}
}

// Execute executes the payment return use case
func (uc *PaymentReturnUseCase) Execute(ctx context.Context, request *dto.PaymentReturnRequest) (*dto.PaymentReturnResponse, error) {
	uc.logger.Info("Processing payment return", map[string]interface{}{
		"order_id":       request.OrderID,
		"oitam_order_id": request.OITAMOrderID,
		"payment_id":     request.PaymentID,
		"payer_id":       request.PayerID,
		"status":         request.Status,
	})

	returnURLs := uc.config.GetReturnURLs()

	// Validate required parameters
	if request.OrderID == "" {
		uc.logger.Error("Missing order ID in payment return", nil, map[string]interface{}{
			"request": request,
		})
		return &dto.PaymentReturnResponse{
			RedirectURL: fmt.Sprintf("%s?error=missing_order_id", returnURLs.Error),
			Status:      "error",
			Message:     "Missing order ID",
		}, nil
	}

	// 1. Verify payment status from OITAM order if available
	if request.OITAMOrderID != "" {
		oitamOrder, err := uc.wooCommerceRepo.GetOITAMOrder(ctx, request.OITAMOrderID)
		if err == nil && oitamOrder.IsPaymentCompleted() {
			// Payment confirmed, update original order
			err = uc.updateOriginalOrderWithPayment(ctx, request, oitamOrder.TransactionID)
			if err != nil {
				uc.logger.Error("Failed to update original order", err, map[string]interface{}{
					"order_id": request.OrderID,
				})
			}

			uc.logger.Info("Payment confirmed via OITAM order", map[string]interface{}{
				"order_id":       request.OrderID,
				"oitam_order_id": request.OITAMOrderID,
				"transaction_id": oitamOrder.TransactionID,
			})

			return &dto.PaymentReturnResponse{
				RedirectURL: fmt.Sprintf("%s?order=%s&payment=confirmed", returnURLs.Success, request.OrderID),
				Status:      "success",
				Message:     "Payment confirmed",
			}, nil
		}
	}

	// 2. Fallback: Update order status based on URL parameters
	if request.PaymentID != "" || request.PayerID != "" {
		err := uc.updateOriginalOrderWithPayment(ctx, request, request.TransactionID)
		if err != nil {
			uc.logger.Error("Failed to update original order (fallback)", err, map[string]interface{}{
				"order_id":   request.OrderID,
				"payment_id": request.PaymentID,
			})
		} else {
			uc.logger.Info("Payment processed via fallback method", map[string]interface{}{
				"order_id":   request.OrderID,
				"payment_id": request.PaymentID,
			})

			return &dto.PaymentReturnResponse{
				RedirectURL: fmt.Sprintf("%s?order=%s&payment=success", returnURLs.Success, request.OrderID),
				Status:      "success",
				Message:     "Payment processed",
			}, nil
		}
	}

	// 3. If we can't confirm payment, redirect to error page
	uc.logger.Warn("Could not verify payment", map[string]interface{}{
		"order_id":       request.OrderID,
		"oitam_order_id": request.OITAMOrderID,
		"payment_id":     request.PaymentID,
	})

	return &dto.PaymentReturnResponse{
		RedirectURL: fmt.Sprintf("%s?order=%s&error=payment_verification_failed", returnURLs.Error, request.OrderID),
		Status:      "error",
		Message:     "Payment verification failed",
	}, nil
}

// updateOriginalOrderWithPayment updates the original order with payment information
func (uc *PaymentReturnUseCase) updateOriginalOrderWithPayment(ctx context.Context, request *dto.PaymentReturnRequest, transactionID string) error {
	// Create payment record
	payment := uc.paymentService.CreatePaymentRecord(
		ctx,
		request.OrderID,
		request.PaymentID,
		request.PayerID,
		entities.Money{}, // Amount will be filled from order
		entities.PaymentStatusCompleted,
	)

	// Set transaction ID
	if transactionID != "" {
		payment.TransactionID = transactionID
	} else if request.PaymentID != "" {
		payment.TransactionID = request.PaymentID
	}

	// Store payment record
	if err := uc.paymentRepo.Create(ctx, payment); err != nil {
		uc.logger.Error("Failed to store payment record", err, map[string]interface{}{
			"payment_id": payment.ID,
			"order_id":   request.OrderID,
		})
		// Continue even if payment record storage fails
	}

	// Update original order
	return uc.wooCommerceRepo.UpdateMagicOrderPayment(ctx, request.OrderID, payment)
}