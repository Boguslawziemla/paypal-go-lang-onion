package handlers

import (
	"net/http"
	"paypal-proxy/internal/application/dto"
	"paypal-proxy/internal/application/services"
	"paypal-proxy/internal/domain/interfaces"

	"github.com/gin-gonic/gin"
)

// PaymentHandler handles payment-related HTTP requests
type PaymentHandler struct {
	orchestrator *services.PaymentOrchestrator
	logger       interfaces.Logger
}

// NewPaymentHandler creates a new payment handler
func NewPaymentHandler(orchestrator *services.PaymentOrchestrator, logger interfaces.Logger) *PaymentHandler {
	return &PaymentHandler{
		orchestrator: orchestrator,
		logger:       logger,
	}
}

// PaymentRedirect handles payment redirect requests
func (h *PaymentHandler) PaymentRedirect(c *gin.Context) {
	orderID := c.Query("orderId")
	if orderID == "" {
		h.respondWithError(c, http.StatusBadRequest, "Missing orderId parameter", nil)
		return
	}

	h.logger.Info("Payment redirect request", map[string]interface{}{
		"order_id": orderID,
		"domain":   c.Request.Host,
	})

	request := &dto.PaymentRedirectRequest{
		OrderID: orderID,
		Domain:  c.Request.Host,
	}

	response, err := h.orchestrator.HandlePaymentRedirect(c.Request.Context(), request)
	if err != nil {
		h.logger.Error("Payment redirect failed", err, map[string]interface{}{
			"order_id": orderID,
		})
		h.respondWithError(c, http.StatusInternalServerError, "Payment redirect failed", err)
		return
	}

	// Redirect to PayPal checkout
	c.Redirect(http.StatusFound, response.RedirectURL)
}

// PayPalReturn handles PayPal return requests
func (h *PaymentHandler) PayPalReturn(c *gin.Context) {
	request := &dto.PaymentReturnRequest{
		OrderID:       c.Query("order_id"),
		OITAMOrderID:  c.Query("oitam_order_id"),
		Status:        c.Query("status"),
		PaymentID:     c.Query("paymentId"),
		PayerID:       c.Query("PayerID"),
		TransactionID: c.Query("transaction_id"),
	}

	h.logger.Info("PayPal return request", map[string]interface{}{
		"order_id":       request.OrderID,
		"oitam_order_id": request.OITAMOrderID,
		"payment_id":     request.PaymentID,
		"status":         request.Status,
	})

	response, err := h.orchestrator.HandlePaymentReturn(c.Request.Context(), request)
	if err != nil {
		h.logger.Error("PayPal return handling failed", err, map[string]interface{}{
			"order_id": request.OrderID,
		})
		// Use fallback error redirect
		c.Redirect(http.StatusFound, "/blad-platnosci?error=return_handler_failed")
		return
	}

	c.Redirect(http.StatusFound, response.RedirectURL)
}

// PayPalCancel handles PayPal cancel requests
func (h *PaymentHandler) PayPalCancel(c *gin.Context) {
	request := &dto.PaymentCancelRequest{
		OrderID:      c.Query("order_id"),
		OITAMOrderID: c.Query("oitam_order_id"),
	}

	h.logger.Info("PayPal cancel request", map[string]interface{}{
		"order_id":       request.OrderID,
		"oitam_order_id": request.OITAMOrderID,
	})

	response, err := h.orchestrator.HandlePaymentCancel(c.Request.Context(), request)
	if err != nil {
		h.logger.Error("PayPal cancel handling failed", err, map[string]interface{}{
			"order_id": request.OrderID,
		})
		// Use fallback error redirect
		c.Redirect(http.StatusFound, "/blad-platnosci?error=cancel_handler_failed")
		return
	}

	c.Redirect(http.StatusFound, response.RedirectURL)
}

// WebhookHandler handles webhook requests
func (h *PaymentHandler) WebhookHandler(c *gin.Context) {
	var request dto.WebhookRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.Error("Failed to parse webhook data", err, map[string]interface{}{
			"content_type": c.GetHeader("Content-Type"),
		})
		h.respondWithError(c, http.StatusBadRequest, "Invalid webhook data", err)
		return
	}

	h.logger.Info("Webhook received", map[string]interface{}{
		"event_type": request.EventType,
		"webhook_id": request.ID,
	})

	response, err := h.orchestrator.HandleWebhook(c.Request.Context(), &request)
	if err != nil {
		h.logger.Error("Webhook processing failed", err, map[string]interface{}{
			"event_type": request.EventType,
			"webhook_id": request.ID,
		})
		h.respondWithError(c, http.StatusInternalServerError, "Webhook processing failed", err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// respondWithError sends an error response
func (h *PaymentHandler) respondWithError(c *gin.Context, statusCode int, message string, err error) {
	errorResponse := dto.ErrorResponse{
		Error:   message,
		Code:    statusCode,
		Message: message,
	}

	if err != nil {
		errorResponse.Message = err.Error()
	}

	c.JSON(statusCode, errorResponse)
}