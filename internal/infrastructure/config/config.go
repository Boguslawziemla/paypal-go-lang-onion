package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"paypal-proxy/internal/domain/interfaces"
)

// Config represents the application configuration
type Config struct {
	Server ServerConfig
	Magic  WooCommerceConfig
	OITAM  WooCommerceConfig
	PayPal PayPalConfig
	Cache  CacheConfig
	DB     DatabaseConfig
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port        string
	Environment string
	LogLevel    string
	BaseURL     string
	Timeout     time.Duration
}

// WooCommerceConfig represents WooCommerce API configuration
type WooCommerceConfig struct {
	URL            string
	ConsumerKey    string
	ConsumerSecret string
	Timeout        time.Duration
	RetryAttempts  int
}

// PayPalConfig represents PayPal configuration
type PayPalConfig struct {
	ClientID     string
	ClientSecret string
	Environment  string // sandbox or live
	WebhookID    string
	Timeout      time.Duration
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	RedisURL    string
	DefaultTTL  time.Duration
	Enabled     bool
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	ConnectionString string
	MaxConnections   int
	Timeout         time.Duration
	Enabled         bool
}

// NewConfig creates a new configuration from environment variables
func NewConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:        getEnv("PORT", "8080"),
			Environment: getEnv("ENVIRONMENT", "development"),
			LogLevel:    getEnv("LOG_LEVEL", "info"),
			BaseURL:     getEnv("BASE_URL", "http://localhost:8080"),
			Timeout:     getDurationEnv("SERVER_TIMEOUT", 30*time.Second),
		},
		Magic: WooCommerceConfig{
			URL:            getEnv("MAGIC_SITE_URL", ""),
			ConsumerKey:    getEnv("MAGIC_CONSUMER_KEY", ""),
			ConsumerSecret: getEnv("MAGIC_CONSUMER_SECRET", ""),
			Timeout:        getDurationEnv("MAGIC_API_TIMEOUT", 30*time.Second),
			RetryAttempts:  getIntEnv("MAGIC_RETRY_ATTEMPTS", 3),
		},
		OITAM: WooCommerceConfig{
			URL:            getEnv("OITAM_SITE_URL", ""),
			ConsumerKey:    getEnv("OITAM_CONSUMER_KEY", ""),
			ConsumerSecret: getEnv("OITAM_CONSUMER_SECRET", ""),
			Timeout:        getDurationEnv("OITAM_API_TIMEOUT", 30*time.Second),
			RetryAttempts:  getIntEnv("OITAM_RETRY_ATTEMPTS", 3),
		},
		PayPal: PayPalConfig{
			ClientID:     getEnv("PAYPAL_CLIENT_ID", ""),
			ClientSecret: getEnv("PAYPAL_CLIENT_SECRET", ""),
			Environment:  getEnv("PAYPAL_ENVIRONMENT", "sandbox"),
			WebhookID:    getEnv("PAYPAL_WEBHOOK_ID", ""),
			Timeout:      getDurationEnv("PAYPAL_TIMEOUT", 30*time.Second),
		},
		Cache: CacheConfig{
			RedisURL:    getEnv("REDIS_URL", ""),
			DefaultTTL:  getDurationEnv("CACHE_DEFAULT_TTL", 15*time.Minute),
			Enabled:     getBoolEnv("CACHE_ENABLED", false),
		},
		DB: DatabaseConfig{
			ConnectionString: getEnv("DATABASE_URL", ""),
			MaxConnections:   getIntEnv("DB_MAX_CONNECTIONS", 10),
			Timeout:         getDurationEnv("DB_TIMEOUT", 30*time.Second),
			Enabled:         getBoolEnv("DATABASE_ENABLED", false),
		},
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	var errors []string

	// Validate server config
	if c.Server.Port == "" {
		errors = append(errors, "PORT is required")
	}

	if c.Server.BaseURL == "" {
		errors = append(errors, "BASE_URL is required")
	}

	// Validate Magic (MagicSpore) config
	if c.Magic.URL == "" {
		errors = append(errors, "MAGIC_SITE_URL is required")
	}

	if c.Magic.ConsumerKey == "" {
		errors = append(errors, "MAGIC_CONSUMER_KEY is required")
	}

	if c.Magic.ConsumerSecret == "" {
		errors = append(errors, "MAGIC_CONSUMER_SECRET is required")
	}

	// Validate OITAM config
	if c.OITAM.URL == "" {
		errors = append(errors, "OITAM_SITE_URL is required")
	}

	if c.OITAM.ConsumerKey == "" {
		errors = append(errors, "OITAM_CONSUMER_KEY is required")
	}

	if c.OITAM.ConsumerSecret == "" {
		errors = append(errors, "OITAM_CONSUMER_SECRET is required")
	}

	// PayPal validation (optional for development)
	if c.Server.Environment == "production" {
		if c.PayPal.ClientID == "" {
			errors = append(errors, "PAYPAL_CLIENT_ID is required for production")
		}

		if c.PayPal.ClientSecret == "" {
			errors = append(errors, "PAYPAL_CLIENT_SECRET is required for production")
		}
	}

	// Validate PayPal environment
	if c.PayPal.Environment != "sandbox" && c.PayPal.Environment != "live" {
		errors = append(errors, "PAYPAL_ENVIRONMENT must be 'sandbox' or 'live'")
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errors, ", "))
	}

	return nil
}

// GetServerConfig returns server configuration
func (c *Config) GetServerConfig() interfaces.ServerConfig {
	return &c.Server
}

// GetMagicSporeConfig returns Magic (MagicSpore) configuration
func (c *Config) GetMagicSporeConfig() interfaces.MagicSporeConfig {
	return interfaces.MagicSporeConfig{
		APIURL:        c.Magic.URL,
		ConsumerKey:   c.Magic.ConsumerKey,
		ConsumerSecret: c.Magic.ConsumerSecret,
	}
}

// GetOITAMConfig returns OITAM configuration
func (c *Config) GetOITAMConfig() interfaces.OITAMConfig {
	return interfaces.OITAMConfig{
		APIURL:        c.OITAM.URL,
		ConsumerKey:   c.OITAM.ConsumerKey,
		ConsumerSecret: c.OITAM.ConsumerSecret,
		CheckoutURL:   c.OITAM.URL + "/checkout",
	}
}

// GetReturnURLs returns the configured return URLs
func (c *Config) GetReturnURLs() interfaces.ReturnURLsConfig {
	successURL := getEnv("SUCCESS_RETURN_URL", "https://magicspore.com/dziekujemy")
	cancelURL := getEnv("CANCEL_RETURN_URL", "https://magicspore.com/koszyk")
	errorURL := getEnv("ERROR_RETURN_URL", "https://magicspore.com/blad-platnosci")
	
	return interfaces.ReturnURLsConfig{
		Success: successURL,
		Cancel:  cancelURL,
		Error:   errorURL,
	}
}

// GetPayPalConfig returns PayPal configuration
func (c *Config) GetPayPalConfig() PayPalConfig {
	return c.PayPal
}

// GetCacheConfig returns cache configuration
func (c *Config) GetCacheConfig() CacheConfig {
	return c.Cache
}

// GetDatabaseConfig returns database configuration
func (c *Config) GetDatabaseConfig() DatabaseConfig {
	return c.DB
}

// IsDevelopment checks if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

// IsProduction checks if running in production mode
func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

// GetWebhookSecret returns the webhook secret for signature validation
func (c *Config) GetWebhookSecret() string {
	return getEnv("WEBHOOK_SECRET", "default-webhook-secret")
}

// GetEncryptionKey returns the encryption key for sensitive data
func (c *Config) GetEncryptionKey() string {
	return getEnv("ENCRYPTION_KEY", "default-encryption-key-change-me")
}

// Interface implementations for ServerConfig
func (s *ServerConfig) GetPort() string {
	return s.Port
}

func (s *ServerConfig) GetEnvironment() string {
	return s.Environment
}

func (s *ServerConfig) GetLogLevel() string {
	return s.LogLevel
}

func (s *ServerConfig) GetBaseURL() string {
	return s.BaseURL
}

func (s *ServerConfig) GetTimeout() time.Duration {
	return s.Timeout
}

// Helper functions for environment variable parsing

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getIntEnv gets an integer environment variable with a default value
func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getBoolEnv gets a boolean environment variable with a default value
func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// getDurationEnv gets a duration environment variable with a default value
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// GetCORSConfig returns CORS configuration
func (c *Config) GetCORSConfig() map[string]interface{} {
	allowedOrigins := getEnv("CORS_ALLOWED_ORIGINS", "*")
	allowedMethods := getEnv("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,OPTIONS")
	allowedHeaders := getEnv("CORS_ALLOWED_HEADERS", "Content-Type,Authorization,X-Requested-With")

	return map[string]interface{}{
		"allowed_origins":    strings.Split(allowedOrigins, ","),
		"allowed_methods":    strings.Split(allowedMethods, ","),
		"allowed_headers":    strings.Split(allowedHeaders, ","),
		"allow_credentials": getBoolEnv("CORS_ALLOW_CREDENTIALS", false),
		"max_age":           getIntEnv("CORS_MAX_AGE", 86400),
	}
}