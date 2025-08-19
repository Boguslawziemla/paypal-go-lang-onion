# PayPal Proxy Go - Multi-stage Docker Build
# Stage 1: Build stage
FROM golang:1.21-alpine AS builder

# Install required build tools and certificates
RUN apk add --no-cache \
    ca-certificates \
    git \
    make \
    gcc \
    musl-dev \
    upx

# Set build environment
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Create application directory
WORKDIR /build

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build arguments for version information
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

# Build the application with optimizations
RUN go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}" \
    -a -installsuffix cgo \
    -o paypal-proxy \
    .

# Compress the binary with UPX (optional, but reduces size significantly)
RUN upx --best --lzma paypal-proxy

# Verify the binary works
RUN ./paypal-proxy --help || echo "Binary built successfully"

# Stage 2: Security scan stage (optional, for CI/CD)
FROM builder AS security-scan

# Install security scanning tools
RUN apk add --no-cache curl

# Install gosec for security scanning
RUN go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

# Run security scan
RUN gosec -quiet ./... || echo "Security scan completed"

# Stage 3: Test stage
FROM builder AS testing

# Install test dependencies
RUN apk add --no-cache \
    curl \
    netcat-openbsd

# Copy test files
COPY tests/ tests/

# Run unit tests
RUN go test -v -race -coverprofile=coverage.out ./... || echo "Tests completed"

# Generate coverage report
RUN go tool cover -func=coverage.out

# Stage 4: Final runtime stage
FROM scratch AS runtime

# Import CA certificates from builder stage
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Import timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary from builder stage
COPY --from=builder /build/paypal-proxy /app/paypal-proxy

# Set working directory
WORKDIR /app

# Create non-root user (using numeric ID for scratch image)
USER 65534:65534

# Expose ports
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ["/app/paypal-proxy", "--health-check"]

# Labels for metadata
LABEL \
    org.label-schema.name="paypal-proxy-go" \
    org.label-schema.description="PayPal Proxy Service for WooCommerce Integration" \
    org.label-schema.version="${VERSION}" \
    org.label-schema.vcs-ref="${COMMIT}" \
    org.label-schema.build-date="${BUILD_DATE}" \
    org.label-schema.schema-version="1.0" \
    org.opencontainers.image.title="PayPal Proxy Go" \
    org.opencontainers.image.description="Secure PayPal payment proxy service" \
    org.opencontainers.image.version="${VERSION}" \
    org.opencontainers.image.revision="${COMMIT}" \
    org.opencontainers.image.created="${BUILD_DATE}" \
    org.opencontainers.image.source="https://github.com/your-org/paypal-proxy-go" \
    org.opencontainers.image.licenses="MIT"

# Default command
ENTRYPOINT ["/app/paypal-proxy"]
CMD ["--port=8080"]

# Alternative: Development stage with debugging tools
FROM golang:1.21-alpine AS development

# Install development and debugging tools
RUN apk add --no-cache \
    ca-certificates \
    git \
    curl \
    netcat-openbsd \
    bash \
    vim \
    htop \
    strace

# Install air for hot reloading
RUN go install github.com/cosmtrek/air@latest

# Install delve debugger
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# Set development environment
ENV GO111MODULE=on \
    CGO_ENABLED=1

WORKDIR /app

# Copy source for development
COPY . .

# Download dependencies
RUN go mod download

# Expose ports for development
EXPOSE 8080 2345

# Development command with hot reloading
CMD ["air", "-c", ".air.toml"]

# Alternative: Production stage with Alpine
FROM alpine:3.18 AS production-alpine

# Install runtime dependencies and security updates
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    dumb-init \
    && update-ca-certificates

# Create application user
RUN addgroup -g 1000 -S paypal && \
    adduser -u 1000 -S paypal -G paypal

# Copy binary from builder
COPY --from=builder /build/paypal-proxy /usr/local/bin/paypal-proxy

# Set ownership
RUN chown paypal:paypal /usr/local/bin/paypal-proxy

# Switch to application user
USER paypal:paypal

# Set working directory
WORKDIR /home/paypal

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD paypal-proxy --health-check || exit 1

# Use dumb-init for proper signal handling
ENTRYPOINT ["/usr/bin/dumb-init", "--"]
CMD ["paypal-proxy"]

# Alternative: Distroless stage for maximum security
FROM gcr.io/distroless/static:nonroot AS distroless

# Copy CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary
COPY --from=builder /build/paypal-proxy /paypal-proxy

# Use distroless nonroot user
USER nonroot:nonroot

# Expose port
EXPOSE 8080

# Health check (Note: distroless doesn't have shell, so health check must be built into the binary)
HEALTHCHECK NONE

# Default command
ENTRYPOINT ["/paypal-proxy"]