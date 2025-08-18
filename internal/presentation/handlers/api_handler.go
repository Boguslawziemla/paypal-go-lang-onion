package handlers

import (
	"net/http"
	"paypal-proxy/internal/application/dto"
	"paypal-proxy/internal/domain/interfaces"

	"github.com/gin-gonic/gin"
)

// APIHandler handles API requests
type APIHandler struct {
	wooCommerceRepo interfaces.WooCommerceRepository
	logger          interfaces.Logger
}

// NewAPIHandler creates a new API handler
func NewAPIHandler(wooCommerceRepo interfaces.WooCommerceRepository, logger interfaces.Logger) *APIHandler {
	return &APIHandler{
		wooCommerceRepo: wooCommerceRepo,
		logger:          logger,
	}
}

// GetOrder retrieves order information
func (h *APIHandler) GetOrder(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		h.respondWithError(c, http.StatusBadRequest, "Order ID is required", nil)
		return
	}

	h.logger.Info("API get order request", map[string]interface{}{
		"order_id": orderID,
	})

	order, err := h.wooCommerceRepo.GetMagicOrder(c.Request.Context(), orderID)
	if err != nil {
		h.logger.Error("Failed to get order", err, map[string]interface{}{
			"order_id": orderID,
		})
		h.respondWithError(c, http.StatusNotFound, "Order not found", err)
		return
	}

	c.JSON(http.StatusOK, order)
}

// GetOrderStatus retrieves order status
func (h *APIHandler) GetOrderStatus(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		h.respondWithError(c, http.StatusBadRequest, "Order ID is required", nil)
		return
	}

	h.logger.Info("API get order status request", map[string]interface{}{
		"order_id": orderID,
	})

	order, err := h.wooCommerceRepo.GetMagicOrder(c.Request.Context(), orderID)
	if err != nil {
		h.logger.Error("Failed to get order status", err, map[string]interface{}{
			"order_id": orderID,
		})
		h.respondWithError(c, http.StatusNotFound, "Order not found", err)
		return
	}

	response := dto.OrderStatusResponse{
		OrderID:       orderID,
		Status:        string(order.Status),
		Total:         order.Total.Amount,
		Currency:      order.Currency,
		PaymentMethod: order.PaymentMethod,
		DateCreated:   order.DateCreated,
		DatePaid:      order.DatePaid,
		TransactionID: order.TransactionID,
	}

	c.JSON(http.StatusOK, response)
}

// CreateOrder handles order creation (for testing)
func (h *APIHandler) CreateOrder(c *gin.Context) {
	h.logger.Info("API create order request (not implemented)", map[string]interface{}{
		"client_ip": c.ClientIP(),
	})

	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "Order creation not implemented",
		"status":  "not_implemented",
	})
}

// UpdateOrder handles order updates (for testing)
func (h *APIHandler) UpdateOrder(c *gin.Context) {
	orderID := c.Param("id")

	h.logger.Info("API update order request (not implemented)", map[string]interface{}{
		"order_id":  orderID,
		"client_ip": c.ClientIP(),
	})

	c.JSON(http.StatusNotImplemented, gin.H{
		"message":  "Order update not implemented",
		"order_id": orderID,
		"status":   "not_implemented",
	})
}

// respondWithError sends an error response
func (h *APIHandler) respondWithError(c *gin.Context, statusCode int, message string, err error) {
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