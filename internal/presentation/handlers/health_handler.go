package handlers

import (
	"net/http"
	"paypal-proxy/internal/application/dto"
	"paypal-proxy/internal/domain/interfaces"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	logger    interfaces.Logger
	startTime time.Time
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(logger interfaces.Logger) *HealthHandler {
	return &HealthHandler{
		logger:    logger,
		startTime: time.Now(),
	}
}

// HealthCheck handles health check requests
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	uptime := time.Since(h.startTime)

	response := dto.HealthResponse{
		Status:    "OK",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Uptime:    uptime.String(),
	}

	h.logger.Debug("Health check requested", map[string]interface{}{
		"uptime":    uptime.String(),
		"client_ip": c.ClientIP(),
	})

	c.JSON(http.StatusOK, response)
}