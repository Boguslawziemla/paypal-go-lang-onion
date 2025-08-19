package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"paypal-proxy/internal/domain/interfaces"
	"time"
)

// HTTPClient provides enhanced HTTP client functionality
type HTTPClient struct {
	client *http.Client
	logger interfaces.Logger
}

// HTTPClientConfig holds HTTP client configuration
type HTTPClientConfig struct {
	Timeout         time.Duration
	MaxIdleConns    int
	MaxConnsPerHost int
	IdleTimeout     time.Duration
	SkipTLSVerify   bool
	EnableRetries   bool
	MaxRetries      int
	RetryDelay      time.Duration
}

// NewHTTPClient creates a new enhanced HTTP client
func NewHTTPClient(config HTTPClientConfig, logger interfaces.Logger) *HTTPClient {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.SkipTLSVerify,
		},
		MaxIdleConns:        config.MaxIdleConns,
		MaxIdleConnsPerHost: config.MaxConnsPerHost,
		IdleConnTimeout:     config.IdleTimeout,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	client := &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
	}

	return &HTTPClient{
		client: client,
		logger: logger,
	}
}

// NewDefaultHTTPClient creates a default HTTP client with secure settings
func NewDefaultHTTPClient(logger interfaces.Logger) *HTTPClient {
	return NewHTTPClient(HTTPClientConfig{
		Timeout:         30 * time.Second,
		MaxIdleConns:    10,
		MaxConnsPerHost: 10,
		IdleTimeout:     30 * time.Second,
		SkipTLSVerify:   false, // Always verify SSL in production
		EnableRetries:   true,
		MaxRetries:      3,
		RetryDelay:      time.Second,
	}, logger)
}

// DoRequest executes an HTTP request with logging and error handling
func (h *HTTPClient) DoRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	start := time.Now()
	
	h.logger.Debug("HTTP request starting", map[string]interface{}{
		"method": req.Method,
		"url":    req.URL.String(),
		"host":   req.Host,
	})

	resp, err := h.client.Do(req.WithContext(ctx))
	duration := time.Since(start)

	if err != nil {
		h.logger.Error("HTTP request failed", err, map[string]interface{}{
			"method":   req.Method,
			"url":      req.URL.String(),
			"duration": duration.String(),
		})
		return nil, err
	}

	h.logger.Info("HTTP request completed", map[string]interface{}{
		"method":      req.Method,
		"url":         req.URL.String(),
		"status_code": resp.StatusCode,
		"status":      resp.Status,
		"duration":    duration.String(),
	})

	return resp, nil
}

// DoRequestWithRetry executes an HTTP request with retry logic
func (h *HTTPClient) DoRequestWithRetry(ctx context.Context, req *http.Request, maxRetries int, retryDelay time.Duration) (*http.Response, error) {
	var lastErr error
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * retryDelay):
			}

			h.logger.Debug("Retrying HTTP request", map[string]interface{}{
				"attempt": attempt,
				"url":     req.URL.String(),
			})
		}

		// Clone request for retry
		reqClone := req.Clone(ctx)
		
		resp, err := h.DoRequest(ctx, reqClone)
		if err != nil {
			lastErr = err
			if attempt == maxRetries {
				break
			}
			continue
		}

		// Don't retry on certain status codes
		if resp.StatusCode == http.StatusNotFound ||
		   resp.StatusCode == http.StatusUnauthorized ||
		   resp.StatusCode == http.StatusForbidden {
			return resp, nil
		}

		// Retry on server errors
		if resp.StatusCode >= 500 && attempt < maxRetries {
			resp.Body.Close()
			lastErr = fmt.Errorf("server error: %d %s", resp.StatusCode, resp.Status)
			continue
		}

		return resp, nil
	}

	return nil, lastErr
}

// Get performs a GET request
func (h *HTTPClient) Get(ctx context.Context, url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w", err)
	}

	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return h.DoRequest(ctx, req)
}

// Post performs a POST request
func (h *HTTPClient) Post(ctx context.Context, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create POST request: %w", err)
	}

	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return h.DoRequest(ctx, req)
}

// Put performs a PUT request
func (h *HTTPClient) Put(ctx context.Context, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "PUT", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create PUT request: %w", err)
	}

	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return h.DoRequest(ctx, req)
}

// Delete performs a DELETE request
func (h *HTTPClient) Delete(ctx context.Context, url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create DELETE request: %w", err)
	}

	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return h.DoRequest(ctx, req)
}

// Close closes the HTTP client and cleans up resources
func (h *HTTPClient) Close() {
	if transport, ok := h.client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
}

// AddStandardHeaders adds standard headers to a request
func AddStandardHeaders(req *http.Request, userAgent string) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	} else {
		req.Header.Set("User-Agent", "PayPal-Proxy-Go/1.0")
	}
}

// IsRetryableError checks if an error is retryable
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	
	// Network errors are typically retryable
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout() || netErr.Temporary()
	}
	
	return false
}

// IsRetryableStatusCode checks if an HTTP status code is retryable
func IsRetryableStatusCode(statusCode int) bool {
	switch statusCode {
	case http.StatusInternalServerError,
		 http.StatusBadGateway,
		 http.StatusServiceUnavailable,
		 http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}