package http

import (
	"fmt"
	"net/url"
	"paypal-proxy/internal/domain/entities"
	"paypal-proxy/internal/domain/interfaces"
)

// URLBuilder implements the URLBuilder interface
type URLBuilder struct {
	config interfaces.ConfigService
	logger interfaces.Logger
}

// NewURLBuilder creates a new URL builder
func NewURLBuilder(config interfaces.ConfigService, logger interfaces.Logger) interfaces.URLBuilder {
	return &URLBuilder{
		config: config,
		logger: logger,
	}
}

// BuildCheckoutURL builds a checkout URL for PayPal
func (u *URLBuilder) BuildCheckoutURL(order *entities.Order, returnURL, cancelURL string) string {
	oitamConfig := u.config.GetOITAMConfig()
	
	// Build base checkout URL
	checkoutURL, err := url.Parse(fmt.Sprintf("%s/%d/", oitamConfig.CheckoutURL, order.ID))
	if err != nil {
		u.logger.Error("Failed to parse checkout URL", err, map[string]interface{}{
			"base_url":  oitamConfig.CheckoutURL,
			"order_id":  order.ID,
		})
		// Return a fallback URL
		return fmt.Sprintf("%s/%d/?pay_for_order=true&key=%s", oitamConfig.CheckoutURL, order.ID, order.OrderKey)
	}
	
	// Add query parameters
	params := url.Values{}
	params.Set("pay_for_order", "true")
	params.Set("key", order.OrderKey)
	
	if returnURL != "" {
		params.Set("return_url", returnURL)
	}
	if cancelURL != "" {
		params.Set("cancel_return", cancelURL)
	}
	
	checkoutURL.RawQuery = params.Encode()
	
	finalURL := checkoutURL.String()
	
	u.logger.Info("Built checkout URL", map[string]interface{}{
		"order_id":    order.ID,
		"checkout_url": finalURL,
		"return_url":  returnURL,
		"cancel_url":  cancelURL,
	})
	
	return finalURL
}

// BuildReturnURL builds a return URL after payment
func (u *URLBuilder) BuildReturnURL(baseURL string, orderID string, paymentID string, status string) string {
	returnURL, err := url.Parse(fmt.Sprintf("%s/paypal-return", baseURL))
	if err != nil {
		u.logger.Error("Failed to parse return URL", err, map[string]interface{}{
			"base_url": baseURL,
			"order_id": orderID,
		})
		return fmt.Sprintf("%s/paypal-return?order_id=%s", baseURL, orderID)
	}
	
	params := url.Values{}
	params.Set("order_id", orderID)
	
	if paymentID != "" {
		params.Set("oitam_order_id", paymentID)
	}
	if status != "" {
		params.Set("status", status)
	}
	
	returnURL.RawQuery = params.Encode()
	
	finalURL := returnURL.String()
	
	u.logger.Debug("Built return URL", map[string]interface{}{
		"order_id":   orderID,
		"payment_id": paymentID,
		"status":     status,
		"return_url": finalURL,
	})
	
	return finalURL
}

// BuildCancelURL builds a cancel URL
func (u *URLBuilder) BuildCancelURL(baseURL string, orderID string) string {
	cancelURL, err := url.Parse(fmt.Sprintf("%s/paypal-cancel", baseURL))
	if err != nil {
		u.logger.Error("Failed to parse cancel URL", err, map[string]interface{}{
			"base_url": baseURL,
			"order_id": orderID,
		})
		return fmt.Sprintf("%s/paypal-cancel?order_id=%s", baseURL, orderID)
	}
	
	params := url.Values{}
	params.Set("order_id", orderID)
	
	cancelURL.RawQuery = params.Encode()
	
	finalURL := cancelURL.String()
	
	u.logger.Debug("Built cancel URL", map[string]interface{}{
		"order_id":  orderID,
		"cancel_url": finalURL,
	})
	
	return finalURL
}

// BuildWebhookURL builds a webhook URL
func (u *URLBuilder) BuildWebhookURL(baseURL string) string {
	webhookURL := fmt.Sprintf("%s/webhook", baseURL)
	
	u.logger.Debug("Built webhook URL", map[string]interface{}{
		"webhook_url": webhookURL,
	})
	
	return webhookURL
}

// BuildHealthCheckURL builds a health check URL
func (u *URLBuilder) BuildHealthCheckURL(baseURL string) string {
	healthURL := fmt.Sprintf("%s/health", baseURL)
	
	return healthURL
}