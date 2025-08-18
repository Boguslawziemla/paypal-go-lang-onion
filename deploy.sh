#!/bin/bash

echo "🚀 PayPal Proxy Go - Quick Deploy Script"
echo "========================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed. Please install Go 1.21+ first.${NC}"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED_VERSION="1.21"

if ! printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V -C; then
    echo -e "${RED}❌ Go version $GO_VERSION is too old. Requires Go $REQUIRED_VERSION+${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Go version $GO_VERSION detected${NC}"

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    echo -e "${BLUE}📝 Creating environment file...${NC}"
    cp .env.example .env
    echo -e "${YELLOW}⚠️  Please edit .env file with your API keys before running!${NC}"
fi

# Build the application
echo -e "${BLUE}🔨 Building application...${NC}"
go mod tidy
go build -o bin/paypal-proxy .

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Build successful!${NC}"
else
    echo -e "${RED}❌ Build failed!${NC}"
    exit 1
fi

# Check if Docker is available
if command -v docker &> /dev/null; then
    echo -e "${BLUE}🐳 Docker detected. Building Docker image...${NC}"
    docker build -t paypal-proxy-go .
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ Docker image built successfully!${NC}"
        
        echo -e "${BLUE}🚀 Starting with Docker Compose...${NC}"
        docker-compose up -d
        
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}✅ PayPal Proxy is now running!${NC}"
            echo ""
            echo -e "${BLUE}📋 Service Information:${NC}"
            echo "🔗 Health Check: http://localhost:8080/health"
            echo "🔗 API Endpoint: http://localhost:8080/redirect?orderId=123"
            echo ""
            echo -e "${BLUE}📊 To check status:${NC}"
            echo "docker-compose ps"
            echo "docker-compose logs -f paypal-proxy"
            echo ""
            echo -e "${BLUE}🛑 To stop:${NC}"
            echo "docker-compose down"
        else
            echo -e "${RED}❌ Failed to start with Docker Compose${NC}"
            exit 1
        fi
    else
        echo -e "${RED}❌ Docker build failed!${NC}"
        exit 1
    fi
else
    echo -e "${YELLOW}⚠️  Docker not found. Running locally...${NC}"
    echo -e "${GREEN}✅ Starting PayPal Proxy locally on port 8080...${NC}"
    echo -e "${BLUE}🔗 Health Check: http://localhost:8080/health${NC}"
    echo ""
    ./bin/paypal-proxy
fi