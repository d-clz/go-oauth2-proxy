# Multi-stage Dockerfile for Token Gateway - Production Ready

# Build stage - using Go 1.25 to match your local environment
FROM golang:1.25-alpine3.22 AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy all source code
COPY . .

# Build binary with optimized flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -v \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -trimpath \
    -o token-gateway \
    ./cmd/gateway

# Verify the binary was built
RUN test -f token-gateway && chmod +x token-gateway

# Runtime stage - using latest stable Alpine
FROM alpine:3.22

# Install runtime dependencies and apply security updates
RUN apk --no-cache upgrade && \
    apk --no-cache add ca-certificates tzdata curl && \
    rm -rf /var/cache/apk/*

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/token-gateway .

# Copy configuration files
COPY config.example.yaml ./config.yaml

# Create non-root user with restricted permissions
RUN addgroup -g 1000 gateway && \
    adduser -D -u 1000 -G gateway -s /sbin/nologin gateway && \
    chown -R gateway:gateway /app && \
    chmod 500 /app/token-gateway && \
    chmod 400 /app/config.yaml

# Switch to non-root user
USER gateway

# Expose application port
EXPOSE 8080

# Enhanced health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/healthz || exit 1

# Environment variables
ENV CONFIG_PATH=/app/config.yaml \
    PORT=8080

# Metadata labels
LABEL org.opencontainers.image.title="Token Gateway" \
      org.opencontainers.image.description="OAuth2 Token Gateway Service" \
      security.scan.enabled="true" \
      security.base-image="alpine:3.20" \
      security.golang-version="1.25"

# Run the application
ENTRYPOINT ["/app/token-gateway"]
CMD ["-config", "/app/config.yaml"]