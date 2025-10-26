# Multi-stage Dockerfile for Token Gateway

# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy go mod files
COPY go.mod ./

# Generate go.sum and download dependencies
RUN go mod download && go mod tidy

# Copy all source code
COPY . .

# Build binary with verbose output
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -v \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o token-gateway \
    ./cmd/gateway

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/token-gateway .

# Copy example config (can be overridden at runtime)
COPY config.example.yaml ./config.yaml

# Create non-root user
RUN addgroup -g 1000 gateway && \
    adduser -D -u 1000 -G gateway gateway && \
    chown -R gateway:gateway /app

USER gateway

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

# Set environment variables
ENV CONFIG_PATH=/app/config.yaml

# Run
ENTRYPOINT ["/app/token-gateway"]
CMD ["-config", "/app/config.yaml"]
