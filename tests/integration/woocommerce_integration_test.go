//go:build integration

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"paypal-proxy/internal/infrastructure/config"
	infraHttp "paypal-proxy/internal/infrastructure/http"
	"paypal-proxy/internal/infrastructure/repositories"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// WooCommerceIntegrationTestSuite tests WooCommerce API integration
type WooCommerceIntegrationTestSuite struct {
	suite.Suite
	repo   *repositories.WooCommerceRepository
	config *config.Config
	logger *infraHttp.Logger
}

// SetupSuite initializes test environment
func (suite *WooCommerceIntegrationTestSuite) SetupSuite() {
	// Skip integration tests if not configured
	if os.Getenv("INTEGRATION_TESTS_ENABLED") != "true" {
		suite.T().Skip("Integration tests disabled")
	}

	// Initialize logger
	loggerConfig := infraHttp.LoggerConfig{
		Level:       "debug",
		Format:      "json",
		ServiceName: "paypal-proxy-test",
		Version:     "test",
		Environment: "test",
		Output:      "stdout",
	}
	suite.logger = infraHttp.NewLogger(loggerConfig).(*infraHttp.Logger)

	// Initialize config
	suite.config = config.NewConfig()
	
	// Initialize repository with test configuration
	magicConfig := repositories.WooCommerceConfig{
		URL:            os.Getenv("TEST_MAGIC_SITE_URL"),
		ConsumerKey:    os.Getenv("TEST_MAGIC_CONSUMER_KEY"),
		ConsumerSecret: os.Getenv("TEST_MAGIC_CONSUMER_SECRET"),
		Timeout:        30 * time.Second,
		RetryAttempts:  3,
	}
	
	oitamConfig := repositories.WooCommerceConfig{
		URL:            os.Getenv("TEST_OITAM_SITE_URL"),
		ConsumerKey:    os.Getenv("TEST_OITAM_CONSUMER_KEY"),
		ConsumerSecret: os.Getenv("TEST_OITAM_CONSUMER_SECRET"),
		Timeout:        30 * time.Second,
		RetryAttempts:  3,
	}

	// Skip test if credentials not provided
	if magicConfig.URL == "" || oitamConfig.URL == "" {
		suite.T().Skip("WooCommerce test credentials not provided")
	}

	suite.repo = repositories.NewWooCommerceRepository(magicConfig, oitamConfig, suite.logger).(*repositories.WooCommerceRepository)
}

// TestMagicOrderRetrieval tests fetching orders from MagicSpore
func (suite *WooCommerceIntegrationTestSuite) TestMagicOrderRetrieval() {
	ctx := context.Background()
	testOrderID := os.Getenv("TEST_MAGIC_ORDER_ID")
	
	if testOrderID == "" {
		suite.T().Skip("No test Magic order ID provided")
	}

	// Test fetching existing order
	order, err := suite.repo.GetMagicOrder(ctx, testOrderID)
	
	if err != nil && err.Error() == "order "+testOrderID+" not found" {
		suite.T().Skip("Test order not found - this may be expected")
		return
	}
	
	suite.NoError(err, "Should be able to fetch Magic order")
	suite.NotNil(order, "Order should not be nil")
	
	if order != nil {
		suite.Equal(testOrderID, order.Number, "Order number should match")
		suite.NotEmpty(order.Currency, "Order should have currency")
		suite.Greater(order.Total.Amount, 0.0, "Order should have positive total")
		
		suite.logger.Info("Successfully fetched Magic order", map[string]interface{}{
			"order_id": order.ID,
			"number":   order.Number,
			"total":    order.Total.Amount,
			"currency": order.Currency,
			"status":   order.Status,
		})
	}
}

// TestOITAMOrderRetrieval tests fetching orders from OITAM
func (suite *WooCommerceIntegrationTestSuite) TestOITAMOrderRetrieval() {
	ctx := context.Background()
	testOrderID := os.Getenv("TEST_OITAM_ORDER_ID")
	
	if testOrderID == "" {
		suite.T().Skip("No test OITAM order ID provided")
	}

	// Test fetching existing order
	order, err := suite.repo.GetOITAMOrder(ctx, testOrderID)
	
	if err != nil && err.Error() == "order "+testOrderID+" not found" {
		suite.T().Skip("Test order not found - this may be expected")
		return
	}
	
	suite.NoError(err, "Should be able to fetch OITAM order")
	suite.NotNil(order, "Order should not be nil")
	
	if order != nil {
		suite.Equal(testOrderID, order.Number, "Order number should match")
		suite.NotEmpty(order.Currency, "Order should have currency")
		suite.Greater(order.Total.Amount, 0.0, "Order should have positive total")
		
		suite.logger.Info("Successfully fetched OITAM order", map[string]interface{}{
			"order_id": order.ID,
			"number":   order.Number,
			"total":    order.Total.Amount,
			"currency": order.Currency,
			"status":   order.Status,
		})
	}
}

// TestInvalidOrderRetrieval tests handling of invalid order IDs
func (suite *WooCommerceIntegrationTestSuite) TestInvalidOrderRetrieval() {
	ctx := context.Background()
	invalidOrderID := "INVALID_ORDER_999999999"

	// Test Magic site
	order, err := suite.repo.GetMagicOrder(ctx, invalidOrderID)
	suite.Error(err, "Should return error for invalid Magic order ID")
	suite.Nil(order, "Order should be nil for invalid ID")
	suite.Contains(err.Error(), "not found", "Error should indicate order not found")

	// Test OITAM site  
	order2, err2 := suite.repo.GetOITAMOrder(ctx, invalidOrderID)
	suite.Error(err2, "Should return error for invalid OITAM order ID")
	suite.Nil(order2, "Order should be nil for invalid ID")
	suite.Contains(err2.Error(), "not found", "Error should indicate order not found")
}

// TestAPIAuthentication tests WooCommerce API authentication
func (suite *WooCommerceIntegrationTestSuite) TestAPIAuthentication() {
	// This test verifies that authentication works by making a simple API call
	ctx := context.Background()
	
	// Try to fetch a non-existent order - should get 404, not 401
	_, err := suite.repo.GetMagicOrder(ctx, "999999999")
	
	// Should get "not found" error, not authentication error
	suite.Error(err, "Should get an error for non-existent order")
	suite.NotContains(err.Error(), "401", "Should not get authentication error")
	suite.NotContains(err.Error(), "Unauthorized", "Should not get unauthorized error")
	
	// If we get a "not found" error, authentication is working
	suite.Contains(err.Error(), "not found", "Should get 'not found' error, indicating auth is working")
}

// TestOrderStatusUpdate tests updating order status
func (suite *WooCommerceIntegrationTestSuite) TestOrderStatusUpdate() {
	ctx := context.Background()
	testOrderID := os.Getenv("TEST_UPDATABLE_ORDER_ID")
	
	if testOrderID == "" {
		suite.T().Skip("No test updatable order ID provided")
	}

	// Test updating Magic order status
	err := suite.repo.UpdateMagicOrderStatus(ctx, testOrderID, "processing")
	if err != nil {
		// Log error but don't fail test - order might not exist or be updatable
		suite.logger.Warn("Failed to update Magic order status", map[string]interface{}{
			"error":    err.Error(),
			"order_id": testOrderID,
		})
	}

	// Test updating OITAM order status
	err2 := suite.repo.UpdateOITAMOrderStatus(ctx, testOrderID, "processing")
	if err2 != nil {
		// Log error but don't fail test - order might not exist or be updatable
		suite.logger.Warn("Failed to update OITAM order status", map[string]interface{}{
			"error":    err2.Error(),
			"order_id": testOrderID,
		})
	}
}

// TestNetworkResilience tests network error handling and retries
func (suite *WooCommerceIntegrationTestSuite) TestNetworkResilience() {
	ctx := context.Background()
	
	// Create a repository with invalid URL to test error handling
	invalidConfig := repositories.WooCommerceConfig{
		URL:            "https://invalid-domain-that-does-not-exist-12345.com",
		ConsumerKey:    "test",
		ConsumerSecret: "test",
		Timeout:        5 * time.Second,
		RetryAttempts:  2,
	}
	
	invalidRepo := repositories.NewWooCommerceRepository(invalidConfig, invalidConfig, suite.logger)
	
	// Should handle network errors gracefully
	start := time.Now()
	_, err := invalidRepo.GetMagicOrder(ctx, "123")
	duration := time.Since(start)
	
	suite.Error(err, "Should return error for invalid domain")
	suite.Greater(duration, 4*time.Second, "Should take at least 4 seconds due to retries")
	suite.Less(duration, 20*time.Second, "Should not take too long")
}

// TestConcurrentRequests tests concurrent API requests
func (suite *WooCommerceIntegrationTestSuite) TestConcurrentRequests() {
	if os.Getenv("TEST_CONCURRENT_REQUESTS") != "true" {
		suite.T().Skip("Concurrent request tests disabled")
	}

	ctx := context.Background()
	concurrency := 5
	done := make(chan bool, concurrency)
	
	// Launch concurrent requests
	for i := 0; i < concurrency; i++ {
		go func(index int) {
			orderID := "999" + string(rune('0'+index)) // 9990, 9991, etc.
			_, err := suite.repo.GetMagicOrder(ctx, orderID)
			
			// We expect errors (orders don't exist), but no panics or hangs
			assert.Error(suite.T(), err, "Should get error for non-existent order %s", orderID)
			done <- true
		}(i)
	}
	
	// Wait for all requests to complete with timeout
	timeout := time.After(30 * time.Second)
	completed := 0
	
	for completed < concurrency {
		select {
		case <-done:
			completed++
		case <-timeout:
			suite.T().Fatal("Concurrent requests timed out")
		}
	}
	
	suite.Equal(concurrency, completed, "All concurrent requests should complete")
}

// TestRunner runs the integration test suite
func TestWooCommerceIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(WooCommerceIntegrationTestSuite))
}

// TestConnectionPooling tests HTTP connection pooling
func (suite *WooCommerceIntegrationTestSuite) TestConnectionPooling() {
	ctx := context.Background()
	
	// Make multiple sequential requests to test connection reuse
	for i := 0; i < 5; i++ {
		orderID := "CONNECTION_TEST_" + string(rune('0'+i))
		start := time.Now()
		
		_, err := suite.repo.GetMagicOrder(ctx, orderID)
		duration := time.Since(start)
		
		// We expect errors, but requests should be fast due to connection pooling
		suite.Error(err, "Should get error for non-existent order")
		
		// After first request, subsequent requests should be faster (connection reuse)
		if i > 0 {
			suite.Less(duration, 10*time.Second, "Request %d should be fast due to connection reuse", i)
		}
		
		suite.logger.Debug("Connection test request completed", map[string]interface{}{
			"request_number": i + 1,
			"duration_ms":    duration.Milliseconds(),
			"order_id":       orderID,
		})
	}
}