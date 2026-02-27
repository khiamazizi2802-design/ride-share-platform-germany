# Stage 1: Builder
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a \
    -o /app/compliance-service \
    ./cmd/main.go

# Stage 2: Final minimal image
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata

# Set German timezone
ENV TZ=Europe/Berlin

# Create non-root user and group
RUN addgroup -S compliancegroup && \
    adduser -S -G compliancegroup -H -D complianceuser

# Set working directory
WORKDIR /app

# Copy the compiled binary from builder stage
COPY --from=builder /app/compliance-service /app/compliance-service

# Ensure correct ownership
RUN chown -R complianceuser:compliancegroup /app

# Switch to non-root user
USER complianceuser

# Expose the service port
EXPOSE 8018

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=15s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8018/health || exit 1

# Run the service
ENTRYPOINT ["/app/compliance-service"]