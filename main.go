package main

import (
	"log"
	"os"

	// Application layer
	"paypal-proxy/internal/application/services"
	"paypal-proxy/internal/application/usecases"

	// Domain layer
	domainServices "paypal-proxy/internal/domain/services"

	// Infrastructure layer
	"paypal-proxy/internal/infrastructure/config"
	"paypal-proxy/internal/infrastructure/http"
	"paypal-proxy/internal/infrastructure/repositories"

	// Presentation layer
	"paypal-proxy/internal/presentation/handlers"
	"paypal-proxy/internal/presentation/middleware"

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
		"environment": app.config.GetServerConfig().Environment,
		"version":     "1.0.0",
	})

	if err := app.router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// Application container
type Application struct {
	config *config.Config
	logger *http.Logger
	router *gin.Engine
}

// initializeApplication sets up dependency injection and returns the application
func initializeApplication() (*Application, error) {
	// 1. Infrastructure Layer
	cfg := config.NewConfig()
	serverConfig := cfg.GetServerConfig()
	logger := http.NewLogger(serverConfig.LogLevel).(*http.Logger)

	// Validate configuration
	if configValidator, ok := cfg.(*config.Config); ok {
		if err := configValidator.Validate(); err != nil {
			logger.Error("Configuration validation failed", err, map[string]interface{}{})
			// Continue with warning - some functionality may not work
			logger.Warn("Continuing with invalid configuration - some features may not work", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// Infrastructure services
	wooCommerceRepo := repositories.NewWooCommerceRepository(cfg, logger)
	urlBuilder := http.NewURLBuilder(cfg, logger)

	// 2. Domain Layer
	orderDomainService := domainServices.NewOrderDomainService(logger)
	paymentDomainService := domainServices.NewPaymentDomainService(logger)

	// 3. Application Layer - Use Cases
	redirectUseCase := usecases.NewPaymentRedirectUseCase(
		wooCommerceRepo,
		nil, // PaymentGateway not implemented in this version
		urlBuilder,
		orderDomainService,
		paymentDomainService,
		logger,
		cfg,
	)

	returnUseCase := usecases.NewPaymentReturnUseCase(
		wooCommerceRepo,
		nil, // PaymentRepository not needed for basic implementation
		nil, // PaymentGateway not implemented
		paymentDomainService,
		orderDomainService,
		logger,
		cfg,
	)

	cancelUseCase := usecases.NewPaymentCancelUseCase(
		wooCommerceRepo,
		nil, // PaymentRepository not needed for basic implementation
		paymentDomainService,
		orderDomainService,
		logger,
		cfg,
	)

	webhookUseCase := usecases.NewWebhookUseCase(
		wooCommerceRepo,
		nil, // PaymentRepository not needed for basic implementation
		paymentDomainService,
		orderDomainService,
		logger,
	)

	// Application services
	orchestrator := services.NewPaymentOrchestrator(
		redirectUseCase,
		returnUseCase,
		cancelUseCase,
		webhookUseCase,
		logger,
	)

	// 4. Presentation Layer
	paymentHandler := handlers.NewPaymentHandler(orchestrator, logger)
	healthHandler := handlers.NewHealthHandler(logger)
	apiHandler := handlers.NewAPIHandler(wooCommerceRepo, logger)

	// 5. HTTP Router Setup
	if serverConfig.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Middleware
	router.Use(middleware.ErrorHandling(logger))
	router.Use(middleware.LoggingMiddleware(logger))
	router.Use(gin.Recovery())
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.RateLimit())

	// Routes setup
	setupRoutes(router, paymentHandler, healthHandler, apiHandler)

	return &Application{
		config: cfg.(*config.Config),
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
	// Health check
	router.GET("/health", healthHandler.HealthCheck)

	// Main payment endpoints
	router.GET("/redirect", paymentHandler.PaymentRedirect)
	router.GET("/paypal-return", paymentHandler.PayPalReturn)
	router.GET("/paypal-cancel", paymentHandler.PayPalCancel)
	router.POST("/webhook", paymentHandler.WebhookHandler)

	// API routes
	api := router.Group("/api/v1")
	{
		api.GET("/order/:id", apiHandler.GetOrder)
		api.POST("/order", apiHandler.CreateOrder)
		api.PUT("/order/:id", apiHandler.UpdateOrder)
		api.GET("/status/:id", apiHandler.GetOrderStatus)
	}
}