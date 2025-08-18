# PayPal Proxy Go - Makefile

.PHONY: help build run test clean docker-build docker-run deploy-local

# Default target
help:
	@echo "PayPal Proxy Go - Available commands:"
	@echo "  build         - Build the Go application"
	@echo "  run           - Run the application locally"
	@echo "  test          - Run tests"
	@echo "  clean         - Clean build artifacts"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run with Docker Compose"
	@echo "  deploy-local  - Deploy locally with Docker"
	@echo "  setup         - Initial setup (copy env file)"

# Setup environment
setup:
	@echo "🔧 Setting up environment..."
	@cp .env.example .env
	@echo "✅ Environment file created. Please edit .env with your API keys."

# Build the application
build:
	@echo "🔨 Building PayPal Proxy..."
	@go mod tidy
	@go build -o bin/paypal-proxy .
	@echo "✅ Build completed: bin/paypal-proxy"

# Run locally
run:
	@echo "🚀 Starting PayPal Proxy locally..."
	@go run .

# Run with air for hot reload (development)
dev:
	@echo "🔥 Starting with hot reload..."
	@air

# Test the application
test:
	@echo "🧪 Running tests..."
	@go test -v ./...

# Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	@rm -rf bin/
	@go clean

# Build Docker image
docker-build:
	@echo "🐳 Building Docker image..."
	@docker build -t paypal-proxy-go .
	@echo "✅ Docker image built: paypal-proxy-go"

# Run with Docker Compose
docker-run: docker-build
	@echo "🐳 Starting with Docker Compose..."
	@docker-compose up -d
	@echo "✅ PayPal Proxy running on http://localhost:8080"

# Deploy locally
deploy-local: setup docker-run
	@echo "🚀 Deployed locally!"
	@echo "📋 Next steps:"
	@echo "1. Edit .env file with your API keys"
	@echo "2. Restart: make docker-restart"
	@echo "3. Check health: curl http://localhost:8080/health"

# Restart Docker containers
docker-restart:
	@echo "🔄 Restarting Docker containers..."
	@docker-compose restart

# Stop Docker containers
docker-stop:
	@echo "🛑 Stopping Docker containers..."
	@docker-compose down

# View logs
logs:
	@echo "📄 Viewing logs..."
	@docker-compose logs -f paypal-proxy

# Production deployment
deploy-prod:
	@echo "🚀 Deploying to production..."
	@docker-compose -f docker-compose.prod.yml up -d
	@echo "✅ Production deployment complete!"

# Install development dependencies
install-dev:
	@echo "📦 Installing development dependencies..."
	@go install github.com/cosmtrek/air@latest
	@echo "✅ Development dependencies installed"