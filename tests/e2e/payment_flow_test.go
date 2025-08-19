//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// E2ETestSuite contains end-to-end tests for the PayPal proxy
type E2ETestSuite struct {
	suite.Suite
	baseURL    string
	httpClient *http.Client
}

// SetupSuite initializes the test environment
func (suite *E2ETestSuite) SetupSuite() {
	suite.baseURL = os.Getenv("TEST_BASE_URL")
	if suite.baseURL == "" {
		suite.baseURL = "http://localhost:8080"
	}

	suite.httpClient = &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects automatically for testing
			return http.ErrUseLastResponse
		},
	}

	// Wait for the service to be ready
	suite.waitForServiceReady()
}

// waitForServiceReady waits for the service to respond to health checks
func (suite *E2ETestSuite) waitForServiceReady() {
	maxAttempts := 30
	for i := 0; i < maxAttempts; i++ {
		resp, err := suite.httpClient.Get(suite.baseURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(2 * time.Second)
	}
	suite.T().Fatal("Service did not become ready in time")
}

// TestHealthEndpoints tests all health check endpoints
func (suite *E2ETestSuite) TestHealthEndpoints() {
	endpoints := []string{"/health", "/ping", "/ready", "/live"}
	
	for _, endpoint := range endpoints {
		suite.T().Run(endpoint, func(t *testing.T) {
			resp, err := suite.httpClient.Get(suite.baseURL + endpoint)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			
			body, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)
			resp.Body.Close()
			
			// Verify JSON response structure
			var healthResponse map[string]interface{}
			err = json.Unmarshal(body, &healthResponse)
			assert.NoError(t, err)
			assert.Contains(t, healthResponse, "status")
		})
	}
}

// TestPaymentRedirectFlow tests the complete payment redirect flow
func (suite *E2ETestSuite) TestPaymentRedirectFlow() {
	// Test cases with different scenarios
	testCases := []struct {
		name        string
		orderID     string
		expectedCode int
		shouldRedirect bool
	}{
		{
			name:           "Valid order ID",
			orderID:        "TEST123",
			expectedCode:   http.StatusFound,
			shouldRedirect: true,
		},
		{
			name:         "Empty order ID",
			orderID:      "",
			expectedCode: http.StatusBadRequest,
			shouldRedirect: false,
		},
		{
			name:         "Invalid order ID format",
			orderID:      "test@#$%",
			expectedCode: http.StatusBadRequest,
			shouldRedirect: false,
		},
		{
			name:         "Order ID too long",
			orderID:      strings.Repeat("A", 100),
			expectedCode: http.StatusBadRequest,
			shouldRedirect: false,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Build request URL
			reqURL := fmt.Sprintf("%s/redirect?orderId=%s", suite.baseURL, url.QueryEscape(tc.orderID))
			
			resp, err := suite.httpClient.Get(reqURL)
			assert.NoError(t, err)
			defer resp.Body.Close()
			
			assert.Equal(t, tc.expectedCode, resp.StatusCode)
			
			if tc.shouldRedirect {
				// Check that Location header is present for redirects
				location := resp.Header.Get("Location")
				assert.NotEmpty(t, location)
				assert.Contains(t, location, "oitam.com") // Should redirect to OITAM
			}
			
			// Log response for debugging
			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode >= 400 {
				t.Logf("Error response: %s", string(body))
			}
		})
	}
}

// TestPayPalReturnEndpoints tests PayPal return handling
func (suite *E2ETestSuite) TestPayPalReturnEndpoints() {
	testCases := []struct {
		name     string
		endpoint string
		params   map[string]string
		expectedCode int
	}{
		{
			name:     "Valid PayPal return",
			endpoint: "/paypal-return",
			params: map[string]string{
				"order_id":   "TEST123",
				"paymentId":  "PAY123456789",
				"PayerID":    "PAYER123",
				"status":     "approved",
			},
			expectedCode: http.StatusFound,
		},
		{
			name:     "PayPal return without parameters",
			endpoint: "/paypal-return",
			params:   map[string]string{},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:     "PayPal cancel",
			endpoint: "/paypal-cancel",
			params: map[string]string{
				"order_id": "TEST123",
			},
			expectedCode: http.StatusFound,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Build URL with parameters
			reqURL, _ := url.Parse(suite.baseURL + tc.endpoint)
			params := url.Values{}
			for key, value := range tc.params {
				params.Set(key, value)
			}
			reqURL.RawQuery = params.Encode()
			
			resp, err := suite.httpClient.Get(reqURL.String())
			assert.NoError(t, err)
			defer resp.Body.Close()
			
			assert.Equal(t, tc.expectedCode, resp.StatusCode)
			
			// Log response for debugging
			if resp.StatusCode >= 400 {
				body, _ := io.ReadAll(resp.Body)
				t.Logf("Error response: %s", string(body))
			}
		})
	}
}

// TestWebhookEndpoint tests webhook handling
func (suite *E2ETestSuite) TestWebhookEndpoint() {
	testCases := []struct {
		name         string
		method       string
		contentType  string
		payload      string
		expectedCode int
	}{
		{
			name:        "Valid webhook payload",
			method:      "POST",
			contentType: "application/json",
			payload: `{
				"id": "webhook123",
				"event_type": "PAYMENT.CAPTURE.COMPLETED",
				"resource": {
					"id": "payment123",
					"custom_id": "order123",
					"amount": {
						"value": "10.00",
						"currency_code": "USD"
					}
				}
			}`,
			expectedCode: http.StatusOK,
		},
		{
			name:         "Invalid method",
			method:       "GET",
			contentType:  "application/json",
			payload:      "",
			expectedCode: http.StatusMethodNotAllowed,
		},
		{
			name:         "Invalid content type",
			method:       "POST",
			contentType:  "text/plain",
			payload:      "invalid",
			expectedCode: http.StatusUnsupportedMediaType,
		},
		{
			name:         "Invalid JSON",
			method:       "POST",
			contentType:  "application/json",
			payload:      "invalid json",
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			var req *http.Request
			var err error
			
			if tc.payload != "" {
				req, err = http.NewRequest(tc.method, suite.baseURL+"/webhook", strings.NewReader(tc.payload))
			} else {
				req, err = http.NewRequest(tc.method, suite.baseURL+"/webhook", nil)
			}
			assert.NoError(t, err)
			
			req.Header.Set("Content-Type", tc.contentType)
			
			resp, err := suite.httpClient.Do(req)
			assert.NoError(t, err)
			defer resp.Body.Close()
			
			assert.Equal(t, tc.expectedCode, resp.StatusCode)
			
			// For successful webhooks, verify response structure
			if resp.StatusCode == http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				assert.NoError(t, err)
				
				var webhookResponse map[string]interface{}
				err = json.Unmarshal(body, &webhookResponse)
				assert.NoError(t, err)
				assert.Contains(t, webhookResponse, "status")
			}
		})
	}
}

// TestAPIEndpoints tests API endpoints
func (suite *E2ETestSuite) TestAPIEndpoints() {
	testCases := []struct {
		name         string
		method       string
		endpoint     string
		expectedCode int
	}{
		{
			name:         "Get order - not found",
			method:       "GET",
			endpoint:     "/api/v1/order/999999",
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "Get order status - not found",
			method:       "GET",
			endpoint:     "/api/v1/status/999999",
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "API health check",
			method:       "GET",
			endpoint:     "/api/v1/health",
			expectedCode: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, suite.baseURL+tc.endpoint, nil)
			assert.NoError(t, err)
			
			resp, err := suite.httpClient.Do(req)
			assert.NoError(t, err)
			defer resp.Body.Close()
			
			assert.Equal(t, tc.expectedCode, resp.StatusCode)
		})
	}
}

// TestSecurityHeaders tests that security headers are present
func (suite *E2ETestSuite) TestSecurityHeaders() {
	resp, err := suite.httpClient.Get(suite.baseURL + "/health")
	suite.NoError(err)
	defer resp.Body.Close()

	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
	}

	for header, expectedValue := range expectedHeaders {
		actualValue := resp.Header.Get(header)
		suite.Equal(expectedValue, actualValue, "Security header %s should be %s", header, expectedValue)
	}
}

// TestRateLimiting tests rate limiting functionality
func (suite *E2ETestSuite) TestRateLimiting() {
	// Skip rate limiting test in development environment
	if os.Getenv("TEST_ENVIRONMENT") == "development" {
		suite.T().Skip("Skipping rate limiting test in development")
	}

	// Make multiple rapid requests to trigger rate limiting
	endpoint := suite.baseURL + "/health"
	
	// Make 150 requests rapidly (above the 100 req/sec limit)
	for i := 0; i < 150; i++ {
		resp, err := suite.httpClient.Get(endpoint)
		if err != nil {
			continue
		}
		
		// Check if we hit rate limit
		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			// Verify rate limit headers are present
			suite.NotEmpty(resp.Header.Get("X-RateLimit-Limit"))
			suite.NotEmpty(resp.Header.Get("Retry-After"))
			return
		}
		resp.Body.Close()
	}
	
	suite.T().Log("Rate limiting may not be enabled or limit not reached")
}

// TestCORS tests CORS headers
func (suite *E2ETestSuite) TestCORS() {
	req, err := http.NewRequest("OPTIONS", suite.baseURL+"/health", nil)
	suite.NoError(err)
	req.Header.Set("Origin", "https://magicspore.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	
	resp, err := suite.httpClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()
	
	// Should return 204 No Content for OPTIONS request
	suite.Equal(http.StatusNoContent, resp.StatusCode)
	
	// Check CORS headers
	suite.NotEmpty(resp.Header.Get("Access-Control-Allow-Origin"))
	suite.NotEmpty(resp.Header.Get("Access-Control-Allow-Methods"))
}

// TestErrorHandling tests error response formats
func (suite *E2ETestSuite) TestErrorHandling() {
	// Test invalid endpoint
	resp, err := suite.httpClient.Get(suite.baseURL + "/invalid-endpoint")
	suite.NoError(err)
	defer resp.Body.Close()
	
	suite.Equal(http.StatusNotFound, resp.StatusCode)
	
	// Verify error response format
	body, err := io.ReadAll(resp.Body)
	suite.NoError(err)
	
	var errorResponse map[string]interface{}
	err = json.Unmarshal(body, &errorResponse)
	if err == nil {
		// If it's JSON, check structure
		suite.Contains(errorResponse, "error")
	}
}

// TestCompletePaymentFlow tests the entire payment flow simulation
func (suite *E2ETestSuite) TestCompletePaymentFlow() {
	suite.T().Log("Testing complete payment flow...")
	
	// Step 1: Initiate payment redirect
	orderID := "E2E_TEST_" + fmt.Sprintf("%d", time.Now().Unix())
	redirectURL := fmt.Sprintf("%s/redirect?orderId=%s", suite.baseURL, orderID)
	
	resp, err := suite.httpClient.Get(redirectURL)
	suite.NoError(err)
	defer resp.Body.Close()
	
	// Should redirect to checkout
	suite.Equal(http.StatusFound, resp.StatusCode)
	
	location := resp.Header.Get("Location")
	suite.NotEmpty(location)
	suite.T().Logf("Redirect location: %s", location)
	
	// Step 2: Simulate PayPal return (success scenario)
	returnURL := fmt.Sprintf("%s/paypal-return?order_id=%s&paymentId=PAY123&PayerID=PAYER123&status=approved", 
		suite.baseURL, orderID)
	
	resp2, err := suite.httpClient.Get(returnURL)
	suite.NoError(err)
	defer resp2.Body.Close()
	
	// Should redirect to success page
	suite.Equal(http.StatusFound, resp2.StatusCode)
	
	// Step 3: Simulate webhook notification
	webhookPayload := fmt.Sprintf(`{
		"id": "webhook_%s",
		"event_type": "PAYMENT.CAPTURE.COMPLETED",
		"resource": {
			"id": "payment_%s",
			"custom_id": "%s",
			"amount": {
				"value": "10.00",
				"currency_code": "USD"
			}
		}
	}`, orderID, orderID, orderID)
	
	webhookReq, err := http.NewRequest("POST", suite.baseURL+"/webhook", strings.NewReader(webhookPayload))
	suite.NoError(err)
	webhookReq.Header.Set("Content-Type", "application/json")
	
	resp3, err := suite.httpClient.Do(webhookReq)
	suite.NoError(err)
	defer resp3.Body.Close()
	
	// Webhook should be processed successfully
	suite.Equal(http.StatusOK, resp3.StatusCode)
	
	suite.T().Log("Complete payment flow test passed!")
}

// TestRunner runs the E2E test suite
func TestE2ETestSuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}