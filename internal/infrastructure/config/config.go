package config

import (
	"os"
	"paypal-proxy/internal/domain/interfaces"
	"strconv"
)

// Config implements the ConfigService interface
type Config struct {
	environment   string
	port          string
	logLevel      string
	apiTimeout    int
	rateLimit     int
	magicSpore    interfaces.MagicSporeConfig
	oitam         interfaces.OITAMConfig
	returnURLs    interfaces.ReturnURLsConfig
}

// NewConfig creates a new configuration instance
func NewConfig() interfaces.ConfigService {
	return &Config{
		environment: getEnv("ENVIRONMENT", "development"),
		port:        getEnv("PORT", "8080"),
		logLevel:    getEnv("LOG_LEVEL", "info"),
		apiTimeout:  getEnvInt("API_TIMEOUT", 30),
		rateLimit:   getEnvInt("RATE_LIMIT", 100),
		
		magicSpore: interfaces.MagicSporeConfig{
			APIURL:        getEnv("MAGIC_API_URL", "https://magicspore.com/wp-json/wc/v3/orders"),
			ConsumerKey:   getEnv("MAGIC_CONSUMER_KEY", ""),
			ConsumerSecret: getEnv("MAGIC_CONSUMER_SECRET", ""),
		},
		
		oitam: interfaces.OITAMConfig{
			APIURL:        getEnv("OITAM_API_URL", "https://oitam.com/wp-json/wc/v3/orders"),
			ConsumerKey:   getEnv("OITAM_CONSUMER_KEY", ""),
			ConsumerSecret: getEnv("OITAM_CONSUMER_SECRET", ""),
			CheckoutURL:   getEnv("OITAM_CHECKOUT_URL", "https://oitam.com/checkout/order-pay"),
		},
		
		returnURLs: interfaces.ReturnURLsConfig{
			Success: getEnv("RETURN_URL_SUCCESS", "https://magicspore.com/dziekujemy"),
			Cancel:  getEnv("RETURN_URL_CANCEL", "https://magicspore.com/koszyk"),
			Error:   getEnv("RETURN_URL_ERROR", "https://magicspore.com/blad-platnosci"),
		},
	}
}

// GetMagicSporeConfig returns MagicSpore configuration
func (c *Config) GetMagicSporeConfig() interfaces.MagicSporeConfig {
	return c.magicSpore
}

// GetOITAMConfig returns OITAM configuration
func (c *Config) GetOITAMConfig() interfaces.OITAMConfig {
	return c.oitam
}

// GetReturnURLs returns return URL configuration
func (c *Config) GetReturnURLs() interfaces.ReturnURLsConfig {
	return c.returnURLs
}

// GetServerConfig returns server configuration
func (c *Config) GetServerConfig() interfaces.ServerConfig {
	return interfaces.ServerConfig{
		Environment: c.environment,
		Port:        c.port,
		LogLevel:    c.logLevel,
		APITimeout:  c.apiTimeout,
		RateLimit:   c.rateLimit,
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.magicSpore.ConsumerKey == "" {
		return NewValidationError("MAGIC_CONSUMER_KEY is required")
	}
	if c.magicSpore.ConsumerSecret == "" {
		return NewValidationError("MAGIC_CONSUMER_SECRET is required")
	}
	if c.oitam.ConsumerKey == "" {
		return NewValidationError("OITAM_CONSUMER_KEY is required")
	}
	if c.oitam.ConsumerSecret == "" {
		return NewValidationError("OITAM_CONSUMER_SECRET is required")
	}
	return nil
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func NewValidationError(message string) *ValidationError {
	return &ValidationError{Message: message}
}