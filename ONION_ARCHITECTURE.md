# 🧅 PayPal Proxy Go - Onion Architecture Implementation

## Architecture Overview

This project follows **Clean Architecture/Onion Architecture** principles with dependency inversion, ensuring maintainable, testable, and scalable code.

## 🏗️ Architecture Layers

### 1. Domain Layer (Core) 
**Location**: `internal/domain/`
- **Entities**: Business objects (`Order`, `Payment`, `Money`, etc.)
- **Interfaces**: Contracts for external dependencies
- **Services**: Core business logic and rules
- **No dependencies** on external frameworks or infrastructure

```
domain/
├── entities/          # Core business entities
│   ├── order.go      # Order domain entity
│   └── payment.go    # Payment domain entity
├── interfaces/       # Repository and service interfaces
│   ├── repositories.go
│   └── services.go
└── services/         # Domain business logic
    ├── order_service.go
    └── payment_service.go
```

### 2. Application Layer (Use Cases)
**Location**: `internal/application/`
- **Use Cases**: Application-specific business rules
- **DTOs**: Data transfer objects for API communication
- **Services**: Application services and orchestrators
- **Depends only** on Domain layer

```
application/
├── dto/              # Data transfer objects
│   └── payment_dto.go
├── usecases/         # Application use cases
│   ├── payment_redirect_usecase.go
│   ├── payment_return_usecase.go
│   ├── payment_cancel_usecase.go
│   └── webhook_usecase.go
└── services/         # Application services
    └── payment_orchestrator.go
```

### 3. Infrastructure Layer (External)
**Location**: `internal/infrastructure/`
- **Repositories**: Data access implementations
- **HTTP**: External HTTP clients and utilities
- **Config**: Configuration management
- **Depends on** Domain interfaces (implements them)

```
infrastructure/
├── config/           # Configuration implementation
│   └── config.go
├── repositories/     # Repository implementations
│   └── woocommerce_repository.go
└── http/            # HTTP utilities
    ├── logger.go    # Logger implementation
    └── url_builder.go
```

### 4. Presentation Layer (Controllers)
**Location**: `internal/presentation/`
- **Handlers**: HTTP request handlers
- **Middleware**: HTTP middleware
- **Routes**: Route definitions
- **Depends on** Application layer

```
presentation/
├── handlers/         # HTTP handlers
│   ├── payment_handler.go
│   ├── health_handler.go
│   └── api_handler.go
└── middleware/       # HTTP middleware
    └── middleware.go
```

## 🔄 Dependency Flow

```
Presentation → Application → Domain ← Infrastructure
     ↓              ↓          ↑           ↑
  Handlers    Use Cases   Entities   Repositories
     ↓              ↓          ↑           ↑
 Controllers   Services   Interfaces  Implementations
```

### Key Principles:
1. **Inner layers know nothing about outer layers**
2. **Outer layers depend on inner layers**
3. **Interfaces defined in inner layers, implemented in outer layers**
4. **Domain layer has no external dependencies**

## 🎯 Benefits of This Architecture

### ✅ **Testability**
- Easy to unit test domain logic without infrastructure
- Mock interfaces for isolated testing
- Clean separation of concerns

### ✅ **Maintainability** 
- Changes in infrastructure don't affect business logic
- Clear separation of responsibilities
- Easy to understand code structure

### ✅ **Flexibility**
- Easy to swap implementations (database, payment providers, etc.)
- Framework-independent business logic
- Plugin-style architecture

### ✅ **Scalability**
- Horizontal scaling of individual layers
- Microservice-ready architecture
- Independent deployment of components

## 🔧 Implementation Details

### Domain Entities
```go
// Pure business objects with behavior
type Order struct {
    ID     int
    Status OrderStatus
    Total  Money
    // ... other fields
}

// Business logic methods
func (o *Order) IsPaymentCompleted() bool
func (o *Order) ToAnonymousOrder() *Order
```

### Repository Interfaces (Domain)
```go
// Defined in domain, implemented in infrastructure
type WooCommerceRepository interface {
    GetMagicOrder(ctx context.Context, id string) (*entities.Order, error)
    CreateOITAMOrder(ctx context.Context, order *entities.Order) (*entities.Order, error)
}
```

### Use Cases (Application)
```go
// Orchestrates domain services and repositories
type PaymentRedirectUseCase struct {
    wooCommerceRepo interfaces.WooCommerceRepository
    orderService    *services.OrderDomainService
}

func (uc *PaymentRedirectUseCase) Execute(ctx context.Context, request *dto.PaymentRedirectRequest) (*dto.PaymentRedirectResponse, error)
```

### Dependency Injection (Main)
```go
func initializeApplication() (*Application, error) {
    // 1. Infrastructure Layer
    cfg := config.NewConfig()
    logger := http.NewLogger(cfg.GetServerConfig().LogLevel)
    wooCommerceRepo := repositories.NewWooCommerceRepository(cfg, logger)
    
    // 2. Domain Layer
    orderService := domainServices.NewOrderDomainService(logger)
    
    // 3. Application Layer
    redirectUseCase := usecases.NewPaymentRedirectUseCase(wooCommerceRepo, orderService)
    orchestrator := services.NewPaymentOrchestrator(redirectUseCase)
    
    // 4. Presentation Layer
    paymentHandler := handlers.NewPaymentHandler(orchestrator)
}
```

## 🧪 Testing Strategy

### Unit Tests
- **Domain**: Test business logic in isolation
- **Application**: Test use cases with mocked dependencies
- **Infrastructure**: Test repository implementations
- **Presentation**: Test HTTP handlers

### Integration Tests
- Test entire use case flows
- Test repository implementations with real APIs
- Test HTTP endpoints end-to-end

### Example Test Structure
```go
func TestPaymentRedirectUseCase(t *testing.T) {
    // Arrange
    mockRepo := &MockWooCommerceRepository{}
    mockOrderService := &MockOrderService{}
    useCase := NewPaymentRedirectUseCase(mockRepo, mockOrderService)
    
    // Act
    result, err := useCase.Execute(ctx, request)
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, expectedURL, result.RedirectURL)
}
```

## 🔄 Data Flow Example

### Payment Redirect Flow:
1. **HTTP Request** → `PaymentHandler.PaymentRedirect()`
2. **Handler** → `PaymentOrchestrator.HandlePaymentRedirect()`
3. **Orchestrator** → `PaymentRedirectUseCase.Execute()`
4. **Use Case** → `WooCommerceRepository.GetMagicOrder()`
5. **Use Case** → `OrderDomainService.ValidateOrderForPayment()`
6. **Use Case** → `OrderDomainService.CreateAnonymousOrder()`
7. **Use Case** → `WooCommerceRepository.CreateOITAMOrder()`
8. **Use Case** → `URLBuilder.BuildCheckoutURL()`
9. **Handler** → HTTP Redirect Response

## 📚 Key Files and Their Responsibilities

| File | Layer | Responsibility |
|------|-------|----------------|
| `domain/entities/order.go` | Domain | Order business entity and rules |
| `domain/interfaces/repositories.go` | Domain | Repository contracts |
| `domain/services/order_service.go` | Domain | Order business logic |
| `application/usecases/payment_redirect_usecase.go` | Application | Payment redirect orchestration |
| `infrastructure/repositories/woocommerce_repository.go` | Infrastructure | WooCommerce API implementation |
| `presentation/handlers/payment_handler.go` | Presentation | HTTP request handling |
| `main.go` | Main | Dependency injection and startup |

## 🚀 Running the Application

The onion architecture is fully implemented and the application maintains the same external API while providing much better internal structure:

```bash
# Build with new architecture
go build -o bin/paypal-proxy .

# Run with new architecture
./bin/paypal-proxy

# Same endpoints work as before
curl http://localhost:8080/health
curl http://localhost:8080/redirect?orderId=123
```

## ✅ Migration Complete

The PayPal Proxy has been successfully refactored from a simple layered architecture to a proper **Onion Architecture** while maintaining 100% backward compatibility with existing functionality.

### What Changed:
- ✅ **Structure**: Reorganized into proper onion layers
- ✅ **Dependencies**: Proper dependency inversion implemented
- ✅ **Testability**: Much easier to unit test individual components
- ✅ **Maintainability**: Clear separation of concerns
- ✅ **Extensibility**: Easy to add new features and integrations

### What Stayed the Same:
- ✅ **API**: All endpoints work exactly the same
- ✅ **Configuration**: Same environment variables
- ✅ **Deployment**: Same Docker and deployment scripts
- ✅ **Functionality**: Identical payment processing behavior