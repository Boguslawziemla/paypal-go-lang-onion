package services

import (
	"context"
	"paypal-proxy/internal/application/dto"
	"paypal-proxy/internal/application/usecases"
	"paypal-proxy/internal/domain/interfaces"
)

// PaymentOrchestrator orchestrates all payment-related use cases
type PaymentOrchestrator struct {
	redirectUseCase *usecases.PaymentRedirectUseCase
	returnUseCase   *usecases.PaymentReturnUseCase
	cancelUseCase   *usecases.PaymentCancelUseCase
	webhookUseCase  *usecases.WebhookUseCase
	logger          interfaces.Logger
}

// NewPaymentOrchestrator creates a new payment orchestrator
func NewPaymentOrchestrator(
	redirectUseCase *usecases.PaymentRedirectUseCase,
	returnUseCase *usecases.PaymentReturnUseCase,
	cancelUseCase *usecases.PaymentCancelUseCase,
	webhookUseCase *usecases.WebhookUseCase,
	logger interfaces.Logger,
) *PaymentOrchestrator {
	return &PaymentOrchestrator{
		redirectUseCase: redirectUseCase,
		returnUseCase:   returnUseCase,
		cancelUseCase:   cancelUseCase,
		webhookUseCase:  webhookUseCase,
		logger:          logger,
	}
}

// HandlePaymentRedirect handles payment redirect requests
func (po *PaymentOrchestrator) HandlePaymentRedirect(ctx context.Context, request *dto.PaymentRedirectRequest) (*dto.PaymentRedirectResponse, error) {
	po.logger.Info("Orchestrating payment redirect", map[string]interface{}{
		"order_id": request.OrderID,
		"domain":   request.Domain,
	})
	
	return po.redirectUseCase.Execute(ctx, request)
}

// HandlePaymentReturn handles payment return requests
func (po *PaymentOrchestrator) HandlePaymentReturn(ctx context.Context, request *dto.PaymentReturnRequest) (*dto.PaymentReturnResponse, error) {
	po.logger.Info("Orchestrating payment return", map[string]interface{}{
		"order_id":       request.OrderID,
		"oitam_order_id": request.OITAMOrderID,
		"status":         request.Status,
	})
	
	return po.returnUseCase.Execute(ctx, request)
}

// HandlePaymentCancel handles payment cancel requests
func (po *PaymentOrchestrator) HandlePaymentCancel(ctx context.Context, request *dto.PaymentCancelRequest) (*dto.PaymentCancelResponse, error) {
	po.logger.Info("Orchestrating payment cancel", map[string]interface{}{
		"order_id":       request.OrderID,
		"oitam_order_id": request.OITAMOrderID,
	})
	
	return po.cancelUseCase.Execute(ctx, request)
}

// HandleWebhook handles webhook requests
func (po *PaymentOrchestrator) HandleWebhook(ctx context.Context, request *dto.WebhookRequest) (*dto.WebhookResponse, error) {
	po.logger.Info("Orchestrating webhook processing", map[string]interface{}{
		"event_type": request.EventType,
		"webhook_id": request.ID,
	})
	
	return po.webhookUseCase.Execute(ctx, request)
}

// ValidateRequest validates common request parameters
func (po *PaymentOrchestrator) ValidateRequest(request interface{}) error {
	// Implement validation logic here
	// This could use a validation library like go-playground/validator
	return nil
}