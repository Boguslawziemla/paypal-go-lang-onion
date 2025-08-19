package main

import (
	"log"
	"net/http"
	"os"
	"time"

	// Application layer
	"paypal-proxy/internal/application/services"
	"paypal-proxy/internal/application/usecases"

	// Domain layer
	domainServices "paypal-proxy/internal/domain/services"

	// Infrastructure layer
	"paypal-proxy/internal/infrastructure/config"
	infraHttp "paypal-proxy/internal/infrastructure/http"
	"paypal-proxy/internal/infrastructure/repositories"

	// Presentation layer
	"paypal-proxy/internal/presentation/handlers"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Initialize application using dependency injection
	app, err := initializeApplication()
	if err != nil {
		log.Fatal("Failed to initialize application:", err)
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	app.logger.Info("PayPal Proxy Server starting", map[string]interface{}{
		"port":        port,
		"environment": app.config.GetServerConfig().GetEnvironment(),
		"version":     "1.0.0",
		"log_level":   app.config.GetServerConfig().GetLogLevel(),
	})

	if err := app.router.Run(":" + port); err != nil {
		app.logger.Error("Failed to start server", err, map[string]interface{}{
			"port": port,
		})
		log.Fatal("Failed to start server:", err)
	}
}

// Application container
type Application struct {
	config *config.Config
	logger *infraHttp.Logger
	router *gin.Engine
}

// initializeApplication sets up dependency injection and returns the application
func initializeApplication() (*Application, error) {
	// 1. Infrastructure Layer - Configuration
	cfg := config.NewConfig()
	serverConfig := cfg.GetServerConfig()
	
	// Initialize enhanced logger
	loggerConfig := infraHttp.LoggerConfig{
		Level:       serverConfig.GetLogLevel(),
		Format:      "json",
		ServiceName: "paypal-proxy",
		Version:     "1.0.0",
		Environment: serverConfig.GetEnvironment(),
		Output:      "stdout",
	}
	logger := infraHttp.NewLogger(loggerConfig)
	
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		logger.Error("Configuration validation failed", err, map[string]interface{}{})
		logger.Warn("Continuing with potentially invalid configuration", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Infrastructure - HTTP Client
	httpClient := infraHttp.NewDefaultHTTPClient(logger)
	
	// Infrastructure - Repository Layer
	magicConfig := repositories.WooCommerceConfig{
		URL:            cfg.GetMagicSporeConfig().APIURL,
		ConsumerKey:    cfg.GetMagicSporeConfig().ConsumerKey,
		ConsumerSecret: cfg.GetMagicSporeConfig().ConsumerSecret,
		Timeout:        30 * time.Second,
		RetryAttempts:  3,
	}
	
	oitamConfig := repositories.WooCommerceConfig{
		URL:            cfg.GetOITAMConfig().APIURL,
		ConsumerKey:    cfg.GetOITAMConfig().ConsumerKey,
		ConsumerSecret: cfg.GetOITAMConfig().ConsumerSecret,
		Timeout:        30 * time.Second,
		RetryAttempts:  3,
	}
	
	wooCommerceRepo := repositories.NewWooCommerceRepository(magicConfig, oitamConfig, logger)
	urlBuilder := infraHttp.NewURLBuilder(cfg, logger)

	// 2. Domain Layer - Business Logic Services
	orderDomainService := domainServices.NewOrderDomainService(logger)
	paymentDomainService := domainServices.NewPaymentDomainService(logger)

	// 3. Application Layer - Use Cases
	redirectUseCase := usecases.NewPaymentRedirectUseCase(
		wooCommerceRepo,
		urlBuilder,
		orderDomainService,
		paymentDomainService,
		logger,
		cfg,
	)

	returnUseCase := usecases.NewPaymentReturnUseCase(
		wooCommerceRepo,
		paymentDomainService,
		orderDomainService,
		logger,
		cfg,
	)

	cancelUseCase := usecases.NewPaymentCancelUseCase(
		wooCommerceRepo,
		paymentDomainService,
		orderDomainService,
		logger,
		cfg,
	)

	webhookUseCase := usecases.NewWebhookUseCase(
		wooCommerceRepo,
		paymentDomainService,
		orderDomainService,
		logger,
		cfg,
	)

	// Application services - Orchestrator
	orchestrator := services.NewPaymentOrchestrator(
		redirectUseCase,
		returnUseCase,
		cancelUseCase,
		webhookUseCase,
		logger,
	)

	// 4. Presentation Layer - HTTP Handlers
	paymentHandler := handlers.NewPaymentHandler(orchestrator, logger, cfg)
	healthHandler := handlers.NewHealthHandler(logger, cfg)
	apiHandler := handlers.NewAPIHandler(wooCommerceRepo, logger)

	// 5. HTTP Router Setup
	if serverConfig.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	// Enhanced Middleware Stack
	// Recovery middleware
	router.Use(gin.Recovery())
	
	// Security headers
	router.Use(func(c *gin.Context) {
		handler := infraHttp.SecurityHeaders()
		handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Next()
		})).ServeHTTP(c.Writer, c.Request)
	})
	
	// CORS middleware
	corsConfig := cfg.GetCORSConfig()
	router.Use(func(c *gin.Context) {
		handler := infraHttp.CORS(corsConfig)
		handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Next()
		})).ServeHTTP(c.Writer, c.Request)
	})
	
	// Request logging middleware
	router.Use(func(c *gin.Context) {
		handler := infraHttp.RequestLogger(logger)
		handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Next()
		})).ServeHTTP(c.Writer, c.Request)
	})
	
	// Rate limiting (production only)
	if serverConfig.GetEnvironment() == "production" {
		rateLimiter := infraHttp.NewRateLimiter(100, 10, logger) // 100 req/sec, burst 10
		router.Use(func(c *gin.Context) {
			handler := rateLimiter.RateLimit()
			handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c.Next()
			})).ServeHTTP(c.Writer, c.Request)
		})
	}
	
	// Request timeout middleware
	router.Use(func(c *gin.Context) {
		handler := infraHttp.Timeout(30*time.Second, logger)
		handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Next()
		})).ServeHTTP(c.Writer, c.Request)
	})
	
	// Health check middleware
	router.Use(func(c *gin.Context) {
		handler := infraHttp.HealthCheck("/health")
		handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Next()
		})).ServeHTTP(c.Writer, c.Request)
	})

	// Routes setup
	setupRoutes(router, paymentHandler, healthHandler, apiHandler)

	// Cleanup function for graceful shutdown
	defer func() {
		if httpClient != nil {
			httpClient.Close()
		}
	}()

	// Log successful initialization
	logger.Info("Application initialized successfully", map[string]interface{}{
		"environment": serverConfig.GetEnvironment(),
		"log_level":   serverConfig.GetLogLevel(),
		"port":        serverConfig.GetPort(),
		"features": map[string]interface{}{
			"enhanced_http": true,
			"rate_limiting": serverConfig.GetEnvironment() == "production",
			"security_headers": true,
			"request_logging": true,
			"cors": true,
		},
	})

	return &Application{
		config: cfg,
		logger: logger,
		router: router,
	}, nil
}

// setupRoutes configures all application routes
func setupRoutes(
	router *gin.Engine,
	paymentHandler *handlers.PaymentHandler,
	healthHandler *handlers.HealthHandler,
	apiHandler *handlers.APIHandler,
) {
	// Health check endpoints
	health := router.Group("/")
	{
		health.GET("/health", healthHandler.HealthCheck)
		health.GET("/ping", healthHandler.Ping)
		health.GET("/ready", healthHandler.ReadinessCheck)
		health.GET("/live", healthHandler.LivenessCheck)
	}

	// Main payment flow endpoints
	payment := router.Group("/")
	{
		// Primary payment redirect endpoint
		payment.GET("/redirect", paymentHandler.PaymentRedirect)
		
		// PayPal return handlers
		payment.GET("/paypal-return", paymentHandler.PayPalReturn)
		payment.GET("/paypal-cancel", paymentHandler.PayPalCancel)
		
		// Webhook endpoint for PayPal notifications
		payment.POST("/webhook", paymentHandler.WebhookHandler)
		payment.POST("/paypal-webhook", paymentHandler.WebhookHandler) // Alternative endpoint
	}

	// API routes for order management
	api := router.Group("/api/v1")
	{
		// Order endpoints
		api.GET("/order/:id", apiHandler.GetOrder)
		api.POST("/order", apiHandler.CreateOrder)
		api.PUT("/order/:id", apiHandler.UpdateOrder)
		
		// Status endpoints  
		api.GET("/status/:id", apiHandler.GetOrderStatus)
		api.GET("/order/:id/status", apiHandler.GetOrderStatus)
		
		// Health endpoint for API
		api.GET("/health", healthHandler.HealthCheck)
	}

	// Legacy endpoints for backward compatibility
	legacy := router.Group("/")
	{
		legacy.GET("/paypal", paymentHandler.PaymentRedirect) // Legacy redirect
	}
}