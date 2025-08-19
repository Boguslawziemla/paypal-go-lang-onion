package http

import (
	"context"
	"fmt"
	"net/http"
	"paypal-proxy/internal/domain/interfaces"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// SecurityHeaders adds security headers to HTTP responses
func SecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Security headers
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Content-Security-Policy", "default-src 'self'")
			
			// HSTS header for HTTPS
			if r.TLS != nil {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CORS configures Cross-Origin Resource Sharing
func CORS(config map[string]interface{}) func(http.Handler) http.Handler {
	allowedOrigins := make(map[string]bool)
	if origins, ok := config["allowed_origins"].([]string); ok {
		for _, origin := range origins {
			allowedOrigins[origin] = true
		}
	}
	
	allowedMethods := "GET,POST,PUT,DELETE,OPTIONS"
	if methods, ok := config["allowed_methods"].([]string); ok {
		allowedMethods = strings.Join(methods, ",")
	}
	
	allowedHeaders := "Content-Type,Authorization,X-Requested-With"
	if headers, ok := config["allowed_headers"].([]string); ok {
		allowedHeaders = strings.Join(headers, ",")
	}
	
	allowCredentials := false
	if creds, ok := config["allow_credentials"].(bool); ok {
		allowCredentials = creds
	}
	
	maxAge := 86400
	if age, ok := config["max_age"].(int); ok {
		maxAge = age
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			
			// Check if origin is allowed
			if origin != "" {
				if len(allowedOrigins) == 0 || allowedOrigins["*"] || allowedOrigins[origin] {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
			}
			
			w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
			w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
			w.Header().Set("Access-Control-Max-Age", strconv.Itoa(maxAge))
			
			if allowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			
			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// RequestLogger logs HTTP requests and responses
func RequestLogger(logger interfaces.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			
			// Generate request ID
			requestID := generateRequestID()
			ctx := context.WithValue(r.Context(), "request_id", requestID)
			r = r.WithContext(ctx)
			
			// Create response writer wrapper to capture status code
			wrappedWriter := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}
			
			// Log request
			logger.Info("HTTP request started", map[string]interface{}{
				"request_id":   requestID,
				"method":       r.Method,
				"path":         r.URL.Path,
				"query":        r.URL.RawQuery,
				"remote_addr":  r.RemoteAddr,
				"user_agent":   r.UserAgent(),
				"content_type": r.Header.Get("Content-Type"),
			})
			
			// Process request
			next.ServeHTTP(wrappedWriter, r)
			
			duration := time.Since(start)
			
			// Log response
			logLevel := "info"
			if wrappedWriter.statusCode >= 400 {
				logLevel = "error"
			} else if wrappedWriter.statusCode >= 300 {
				logLevel = "warn"
			}
			
			fields := map[string]interface{}{
				"request_id":    requestID,
				"method":        r.Method,
				"path":          r.URL.Path,
				"status_code":   wrappedWriter.statusCode,
				"duration":      duration.String(),
				"duration_ms":   duration.Milliseconds(),
				"response_size": wrappedWriter.size,
			}
			
			switch logLevel {
			case "error":
				logger.Error("HTTP request completed with error", nil, fields)
			case "warn":
				logger.Warn("HTTP request completed with warning", fields)
			default:
				logger.Info("HTTP request completed", fields)
			}
		})
	}
}

// RateLimiter implements rate limiting middleware
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mutex    sync.RWMutex
	rate     rate.Limit
	burst    int
	logger   interfaces.Logger
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerSecond int, burst int, logger interfaces.Logger) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(requestsPerSecond),
		burst:    burst,
		logger:   logger,
	}
}

// RateLimit returns the rate limiting middleware
func (rl *RateLimiter) RateLimit() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client IP
			ip := getClientIP(r)
			
			// Get or create limiter for this IP
			limiter := rl.getLimiter(ip)
			
			// Check if request is allowed
			if !limiter.Allow() {
				rl.logger.Warn("Rate limit exceeded", map[string]interface{}{
					"client_ip": ip,
					"path":      r.URL.Path,
					"method":    r.Method,
				})
				
				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(int(rl.rate)))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("Retry-After", "60")
				
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// getLimiter gets or creates a rate limiter for an IP
func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mutex.RLock()
	limiter, exists := rl.limiters[ip]
	rl.mutex.RUnlock()
	
	if exists {
		return limiter
	}
	
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	// Double-check locking
	if limiter, exists := rl.limiters[ip]; exists {
		return limiter
	}
	
	// Create new limiter
	limiter = rate.NewLimiter(rl.rate, rl.burst)
	rl.limiters[ip] = limiter
	
	return limiter
}

// Recovery middleware recovers from panics and logs them
func Recovery(logger interfaces.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("Panic recovered", fmt.Errorf("panic: %v", err), map[string]interface{}{
						"request_id": r.Context().Value("request_id"),
						"method":     r.Method,
						"path":       r.URL.Path,
						"panic":      err,
					})
					
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error": "Internal server error"}`))
				}
			}()
			
			next.ServeHTTP(w, r)
		})
	}
}

// Timeout middleware adds request timeout
func Timeout(timeout time.Duration, logger interfaces.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			
			r = r.WithContext(ctx)
			
			done := make(chan bool)
			go func() {
				next.ServeHTTP(w, r)
				done <- true
			}()
			
			select {
			case <-done:
				return
			case <-ctx.Done():
				logger.Warn("Request timeout", map[string]interface{}{
					"request_id": r.Context().Value("request_id"),
					"method":     r.Method,
					"path":       r.URL.Path,
					"timeout":    timeout.String(),
				})
				
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusRequestTimeout)
				w.Write([]byte(`{"error": "Request timeout"}`))
			}
		})
	}
}

// Helper types and functions

// responseWriter wraps http.ResponseWriter to capture response details
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriter) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	w.size += n
	return n, err
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Nanosecond())
}

// getClientIP extracts client IP from request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// Use remote address
	ip := r.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}
	
	return ip
}

// HealthCheck middleware provides health check endpoint
func HealthCheck(path string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" && r.URL.Path == path {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": "healthy", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`))
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}