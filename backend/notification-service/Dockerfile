# Stage 1: Builder
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go module files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the binary with optimizations
# CGO_ENABLED=0 for a fully static binary
# -ldflags reduces binary size by stripping debug info
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -extldflags '-static'" \
    -trimpath \
    -o /app/notification-service \
    ./cmd/server/main.go

# Stage 2: Runtime
FROM alpine:latest

# Install runtime dependencies
# ca-certificates: Required for HTTPS calls to notification providers (Twilio, SendGrid, FCM, etc.)
# tzdata: Timezone support for scheduled notifications
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    && update-ca-certificates

# Create a non-root user and group for security
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup -h /app -s /sbin/nologin

# Set working directory
WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /app/notification-service ./notification-service

# Copy any required config or template files (adjust paths as needed)
# COPY --from=builder /app/configs ./configs
# COPY --from=builder /app/templates ./templates

# Set ownership of the app directory to the non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose the application port
EXPOSE 8080

# Health check to verify the service is running and responsive
# Adjust the endpoint path to match your actual health check route
HEALTHCHECK --interval=30s --timeout=10s --start-period=15s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Set environment defaults (can be overridden at runtime)
ENV APP_ENV=production \
    PORT=8080 \
    LOG_LEVEL=info \
    GIN_MODE=release

# Use exec form of ENTRYPOINT for proper signal handling (SIGTERM/SIGINT)
ENTRYPOINT ["/app/notification-service"]
