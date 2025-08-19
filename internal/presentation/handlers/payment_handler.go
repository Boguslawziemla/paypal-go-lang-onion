package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"paypal-proxy/internal/application/dto"
	"paypal-proxy/internal/application/services"
	"paypal-proxy/internal/domain/interfaces"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// PaymentHandler handles payment-related HTTP requests with security features
type PaymentHandler struct {
	orchestrator  *services.PaymentOrchestrator
	logger        interfaces.Logger
	config        interfaces.ConfigService
	orderIDRegex  *regexp.Regexp
	allowedDomains map[string]bool
}

// NewPaymentHandler creates a new payment handler with security features
func NewPaymentHandler(orchestrator *services.PaymentOrchestrator, logger interfaces.Logger, config interfaces.ConfigService) *PaymentHandler {
	// Compile regex for order ID validation (alphanumeric, 1-50 chars)
	orderIDRegex := regexp.MustCompile(`^[a-zA-Z0-9]{1,50}$`)
	
	// Setup allowed domains for security
	allowedDomains := make(map[string]bool)
	allowedDomains["magicspore.com"] = true
	allowedDomains["www.magicspore.com"] = true
	allowedDomains["oitam.com"] = true
	allowedDomains["www.oitam.com"] = true
	allowedDomains["localhost"] = true
	allowedDomains["127.0.0.1"] = true
	
	return &PaymentHandler{
		orchestrator:  orchestrator,
		logger:        logger,
		config:        config,
		orderIDRegex:  orderIDRegex,
		allowedDomains: allowedDomains,
	}
}

// PaymentRedirect handles payment redirect requests with enhanced security
func (h *PaymentHandler) PaymentRedirect(c *gin.Context) {
	// Security: Validate request method
	if c.Request.Method != "GET" {
		h.logSecurityEvent(c, "invalid_method", "Payment redirect must use GET method")
		h.respondWithError(c, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}
	
	// Security: Check referrer if present
	if !h.isValidReferrer(c) {
		h.logSecurityEvent(c, "invalid_referrer", "Request from untrusted referrer")
		h.respondWithError(c, http.StatusForbidden, "Invalid referrer", nil)
		return
	}
	
	orderID := c.Query("orderId")
	if orderID == "" {
		h.respondWithError(c, http.StatusBadRequest, "Missing orderId parameter", nil)
		return
	}
	
	// Security: Validate order ID format
	if !h.validateOrderID(orderID) {
		h.logSecurityEvent(c, "invalid_order_id", fmt.Sprintf("Invalid order ID format: %s", orderID))
		h.respondWithError(c, http.StatusBadRequest, "Invalid order ID format", nil)
		return
	}

	h.logger.Info("Payment redirect request", map[string]interface{}{
		"order_id":     orderID,
		"domain":       c.Request.Host,
		"user_agent":   c.Request.UserAgent(),
		"remote_addr":  c.ClientIP(),
		"referrer":     c.Request.Referer(),
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

// PayPalReturn handles PayPal return requests with enhanced security
func (h *PaymentHandler) PayPalReturn(c *gin.Context) {
	// Security: Validate request method
	if c.Request.Method != "GET" {
		h.logSecurityEvent(c, "invalid_method", "PayPal return must use GET method")
		h.respondWithError(c, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}
	
	// Security: Rate limiting check (handled by middleware but log here too)
	h.logSecurityEvent(c, "paypal_return", "PayPal return request received")
	
	request := &dto.PaymentReturnRequest{
		OrderID:       h.sanitizeInput(c.Query("order_id")),
		OITAMOrderID:  h.sanitizeInput(c.Query("oitam_order_id")),
		Status:        h.sanitizeInput(c.Query("status")),
		PaymentID:     h.sanitizeInput(c.Query("paymentId")),
		PayerID:       h.sanitizeInput(c.Query("PayerID")),
		TransactionID: h.sanitizeInput(c.Query("transaction_id")),
	}
	
	// Security: Validate required fields
	if !h.validatePaymentReturnRequest(request) {
		h.logSecurityEvent(c, "invalid_return_data", "Invalid PayPal return parameters")
		h.respondWithError(c, http.StatusBadRequest, "Invalid return parameters", nil)
		return
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

// PayPalCancel handles PayPal cancel requests with enhanced security
func (h *PaymentHandler) PayPalCancel(c *gin.Context) {
	// Security: Validate request method
	if c.Request.Method != "GET" {
		h.logSecurityEvent(c, "invalid_method", "PayPal cancel must use GET method")
		h.respondWithError(c, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}
	
	h.logSecurityEvent(c, "paypal_cancel", "PayPal cancel request received")
	
	request := &dto.PaymentCancelRequest{
		OrderID:      h.sanitizeInput(c.Query("order_id")),
		OITAMOrderID: h.sanitizeInput(c.Query("oitam_order_id")),
	}
	
	// Security: Validate order ID if provided
	if request.OrderID != "" && !h.validateOrderID(request.OrderID) {
		h.logSecurityEvent(c, "invalid_order_id", fmt.Sprintf("Invalid order ID in cancel: %s", request.OrderID))
		h.respondWithError(c, http.StatusBadRequest, "Invalid order ID format", nil)
		return
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

// WebhookHandler handles webhook requests with enhanced security
func (h *PaymentHandler) WebhookHandler(c *gin.Context) {
	// Security: Validate request method
	if c.Request.Method != "POST" {
		h.logSecurityEvent(c, "invalid_method", "Webhook must use POST method")
		h.respondWithError(c, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}
	
	// Security: Validate Content-Type
	contentType := c.GetHeader("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		h.logSecurityEvent(c, "invalid_content_type", fmt.Sprintf("Invalid content type: %s", contentType))
		h.respondWithError(c, http.StatusUnsupportedMediaType, "Invalid content type", nil)
		return
	}
	
	// Security: Read and validate webhook signature if configured
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("Failed to read webhook body", err, map[string]interface{}{
			"content_length": c.Request.ContentLength,
		})
		h.respondWithError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	
	// Security: Verify webhook signature
	if !h.verifyWebhookSignature(c, body) {
		h.logSecurityEvent(c, "invalid_signature", "Webhook signature verification failed")
		h.respondWithError(c, http.StatusUnauthorized, "Invalid signature", nil)
		return
	}
	
	// Parse webhook data
	var request dto.WebhookRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.Error("Failed to parse webhook data", err, map[string]interface{}{
			"content_type":   contentType,
			"body_length":    len(body),
			"remote_addr":    c.ClientIP(),
		})
		h.respondWithError(c, http.StatusBadRequest, "Invalid webhook data", err)
		return
	}
	
	// Security: Validate webhook event type
	if !h.isValidWebhookEventType(request.EventType) {
		h.logSecurityEvent(c, "invalid_event_type", fmt.Sprintf("Unknown webhook event type: %s", request.EventType))
		h.respondWithError(c, http.StatusBadRequest, "Invalid event type", nil)
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

// Security helper methods

// validateOrderID validates order ID format and content
func (h *PaymentHandler) validateOrderID(orderID string) bool {
	if orderID == "" {
		return false
	}
	
	// Check length (1-50 characters)
	if len(orderID) < 1 || len(orderID) > 50 {
		return false
	}
	
	// Check format using regex (alphanumeric only)
	return h.orderIDRegex.MatchString(orderID)
}

// sanitizeInput sanitizes user input to prevent XSS and injection attacks
func (h *PaymentHandler) sanitizeInput(input string) string {
	// Remove potentially dangerous characters
	input = strings.ReplaceAll(input, "<", "")
	input = strings.ReplaceAll(input, ">", "")
	input = strings.ReplaceAll(input, "'", "")
	input = strings.ReplaceAll(input, "\"", "")
	input = strings.ReplaceAll(input, ";", "")
	input = strings.ReplaceAll(input, "&", "")
	
	// Trim whitespace
	return strings.TrimSpace(input)
}

// isValidReferrer checks if the request referrer is from an allowed domain
func (h *PaymentHandler) isValidReferrer(c *gin.Context) bool {
	referrer := c.Request.Referer()
	if referrer == "" {
		return true // Allow empty referrer for direct access
	}
	
	// Extract domain from referrer
	if strings.HasPrefix(referrer, "http://") {
		referrer = referrer[7:]
	} else if strings.HasPrefix(referrer, "https://") {
		referrer = referrer[8:]
	}
	
	// Get domain part
	parts := strings.Split(referrer, "/")
	if len(parts) > 0 {
		domain := parts[0]
		// Remove port if present
		if colonIndex := strings.Index(domain, ":"); colonIndex != -1 {
			domain = domain[:colonIndex]
		}
		return h.allowedDomains[domain]
	}
	
	return false
}

// validatePaymentReturnRequest validates PayPal return request parameters
func (h *PaymentHandler) validatePaymentReturnRequest(req *dto.PaymentReturnRequest) bool {
	// Order ID is required and must be valid format
	if req.OrderID == "" || !h.validateOrderID(req.OrderID) {
		return false
	}
	
	// Payment ID should be alphanumeric if present
	if req.PaymentID != "" && !regexp.MustCompile(`^[a-zA-Z0-9_-]{1,100}$`).MatchString(req.PaymentID) {
		return false
	}
	
	// Payer ID should be alphanumeric if present
	if req.PayerID != "" && !regexp.MustCompile(`^[a-zA-Z0-9]{1,50}$`).MatchString(req.PayerID) {
		return false
	}
	
	// Status should be one of expected values if present
	if req.Status != "" {
		validStatuses := []string{"approved", "completed", "cancelled", "failed"}
		valid := false
		for _, status := range validStatuses {
			if req.Status == status {
				valid = true
				break
			}
		}
		if !valid {
			return false
		}
	}
	
	return true
}

// verifyWebhookSignature verifies webhook signature for security
func (h *PaymentHandler) verifyWebhookSignature(c *gin.Context, body []byte) bool {
	// Get webhook secret from config
	webhookSecret := h.config.GetWebhookSecret()
	if webhookSecret == "" || webhookSecret == "default-webhook-secret" {
		// If no secret is configured, skip verification (development mode)
		h.logger.Warn("Webhook signature verification skipped - no secret configured", map[string]interface{}{
			"environment": h.config.GetServerConfig().GetEnvironment(),
		})
		return true
	}
	
	// Get signature from header
	signature := c.GetHeader("X-PayPal-Transmission-Sig")
	if signature == "" {
		signature = c.GetHeader("X-Hub-Signature-256")
	}
	
	if signature == "" {
		return false
	}
	
	// Create HMAC hash
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write(body)
	expectedMAC := mac.Sum(nil)
	expectedSignature := "sha256=" + hex.EncodeToString(expectedMAC)
	
	// Compare signatures
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// isValidWebhookEventType validates webhook event types
func (h *PaymentHandler) isValidWebhookEventType(eventType string) bool {
	validEvents := []string{
		"PAYMENT.CAPTURE.COMPLETED",
		"PAYMENT.CAPTURE.DENIED",
		"PAYMENT.CAPTURE.REFUNDED",
		"PAYMENT.CAPTURE.PENDING",
		"CHECKOUT.ORDER.APPROVED",
		"CHECKOUT.ORDER.COMPLETED",
	}
	
	for _, valid := range validEvents {
		if eventType == valid {
			return true
		}
	}
	
	return false
}

// logSecurityEvent logs security-related events
func (h *PaymentHandler) logSecurityEvent(c *gin.Context, eventType, description string) {
	h.logger.Warn("Security event", map[string]interface{}{
		"event_type":   eventType,
		"description":  description,
		"remote_addr":  c.ClientIP(),
		"user_agent":   c.Request.UserAgent(),
		"method":       c.Request.Method,
		"path":         c.Request.URL.Path,
		"referrer":     c.Request.Referer(),
		"timestamp":    time.Now().Unix(),
	})
}

// respondWithError sends an error response with security logging
func (h *PaymentHandler) respondWithError(c *gin.Context, statusCode int, message string, err error) {
	// Add security headers
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-Frame-Options", "DENY")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	
	errorResponse := dto.ErrorResponse{
		Error:     message,
		Code:      statusCode,
		Message:   message,
		Timestamp: time.Now().Unix(),
	}

	// In production, don't expose internal error details
	if h.config.GetServerConfig().GetEnvironment() == "production" && err != nil {
		errorResponse.Message = "Internal server error"
		// Log the actual error internally
		h.logger.Error("Internal error (hidden from response)", err, map[string]interface{}{
			"status_code": statusCode,
			"client_ip":   c.ClientIP(),
		})
	} else if err != nil {
		errorResponse.Message = err.Error()
	}

	c.JSON(statusCode, errorResponse)
}