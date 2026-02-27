# Stage 1: Build
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o analytics-service ./cmd/analytics-service

# Stage 2: Run
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create non-root user and group
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/analytics-service .

# Set ownership of the binary
RUN chown appuser:appgroup /app/analytics-service

# Switch to non-root user
USER appuser

# Expose application port
EXPOSE 8080

# Run the binary
CMD ["/app/analytics-service"]
