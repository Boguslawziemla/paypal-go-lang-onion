package usecases

import (
	"context"
	"fmt"
	"paypal-proxy/internal/application/dto"
	"paypal-proxy/internal/domain/entities"
	"paypal-proxy/internal/domain/interfaces"
	"paypal-proxy/internal/domain/services"
)

// PaymentRedirectUseCase handles the payment redirect use case
type PaymentRedirectUseCase struct {
	wooCommerceRepo interfaces.WooCommerceRepository
	paymentGateway  interfaces.PaymentGateway
	urlBuilder      interfaces.URLBuilder
	orderService    *services.OrderDomainService
	paymentService  *services.PaymentDomainService
	logger          interfaces.Logger
	config          interfaces.ConfigService
}

// NewPaymentRedirectUseCase creates a new payment redirect use case
func NewPaymentRedirectUseCase(
	wooCommerceRepo interfaces.WooCommerceRepository,
	paymentGateway interfaces.PaymentGateway,
	urlBuilder interfaces.URLBuilder,
	orderService *services.OrderDomainService,
	paymentService *services.PaymentDomainService,
	logger interfaces.Logger,
	config interfaces.ConfigService,
) *PaymentRedirectUseCase {
	return &PaymentRedirectUseCase{
		wooCommerceRepo: wooCommerceRepo,
		paymentGateway:  paymentGateway,
		urlBuilder:      urlBuilder,
		orderService:    orderService,
		paymentService:  paymentService,
		logger:          logger,
		config:          config,
	}
}

// Execute executes the payment redirect use case
func (uc *PaymentRedirectUseCase) Execute(ctx context.Context, request *dto.PaymentRedirectRequest) (*dto.PaymentRedirectResponse, error) {
	uc.logger.Info("Starting payment redirect", map[string]interface{}{
		"order_id": request.OrderID,
		"domain":   request.Domain,
	})

	// 1. Fetch original order from MagicSpore
	magicOrder, err := uc.wooCommerceRepo.GetMagicOrder(ctx, request.OrderID)
	if err != nil {
		uc.logger.Error("Failed to fetch MagicSpore order", err, map[string]interface{}{
			"order_id": request.OrderID,
		})
		return nil, fmt.Errorf("failed to fetch original order: %w", err)
	}

	// 2. Validate order for payment processing
	if err := uc.orderService.ValidateOrderForPayment(ctx, magicOrder); err != nil {
		// Check if order is already paid
		if magicOrder.IsPaymentCompleted() {
			uc.logger.Info("Order already paid, redirecting to success", map[string]interface{}{
				"order_id": request.OrderID,
				"status":   magicOrder.Status,
			})
			
			returnURLs := uc.config.GetReturnURLs()
			successURL := fmt.Sprintf("%s?order=%s&already_paid=1", returnURLs.Success, request.OrderID)
			
			return &dto.PaymentRedirectResponse{
				RedirectURL:  successURL,
				OrderID:      request.OrderID,
				Status:       "already_paid",
				Message:      "Order payment already completed",
			}, nil
		}
		
		uc.logger.Error("Order validation failed", err, map[string]interface{}{
			"order_id": request.OrderID,
			"status":   magicOrder.Status,
		})
		return nil, fmt.Errorf("order validation failed: %w", err)
	}

	// 3. Create anonymous proxy order
	anonymousOrder, err := uc.orderService.CreateAnonymousOrder(ctx, magicOrder)
	if err != nil {
		uc.logger.Error("Failed to create anonymous order", err, map[string]interface{}{
			"order_id": request.OrderID,
		})
		return nil, fmt.Errorf("failed to create anonymous order: %w", err)
	}

	// 4. Create proxy order on OITAM
	oitamOrder, err := uc.wooCommerceRepo.CreateOITAMOrder(ctx, anonymousOrder)
	if err != nil {
		uc.logger.Error("Failed to create OITAM order", err, map[string]interface{}{
			"order_id":        request.OrderID,
			"anonymous_order": anonymousOrder.Number,
		})
		return nil, fmt.Errorf("failed to create payment order: %w", err)
	}

	// 5. Build return and cancel URLs
	returnURL := uc.urlBuilder.BuildReturnURL(
		fmt.Sprintf("https://%s", request.Domain),
		request.OrderID,
		fmt.Sprintf("%d", oitamOrder.ID),
		"success",
	)
	
	cancelURL := uc.urlBuilder.BuildCancelURL(
		fmt.Sprintf("https://%s", request.Domain),
		request.OrderID,
	)

	// 6. Build checkout URL
	checkoutURL := uc.urlBuilder.BuildCheckoutURL(oitamOrder, returnURL, cancelURL)

	uc.logger.Info("Payment redirect created successfully", map[string]interface{}{
		"order_id":       request.OrderID,
		"oitam_order_id": oitamOrder.ID,
		"checkout_url":   checkoutURL,
	})

	return &dto.PaymentRedirectResponse{
		RedirectURL:  checkoutURL,
		OrderID:      request.OrderID,
		ProxyOrderID: fmt.Sprintf("%d", oitamOrder.ID),
		Status:       "redirect_created",
		Message:      "Redirect to PayPal checkout created",
	}, nil
}