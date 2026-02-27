# Stage 1: Builder
FROM golang:1.21-alpine AS builder

# Install necessary system dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    && update-ca-certificates

# Create a non-root user for the final image
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid 65532 \
    appuser

# Set working directory
WORKDIR /app

# Copy dependency files first to leverage Docker layer caching
COPY go.mod go.sum ./

# Download and verify dependencies
RUN go mod download && go mod verify

# Copy the full source code
COPY . .

# Build the static binary with security and optimization flags
RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build \
    -ldflags="-w -s -extldflags '-static'" \
    -a \
    -installsuffix cgo \
    -o /app/compliance-audit-service \
    ./cmd/main.go

# Stage 2: Final minimal image
FROM scratch

# Add metadata labels
LABEL maintainer="platform-team@company.com" \
      service="compliance-audit-service" \
      version="1.0.0" \
      description="Go Compliance and Audit Microservice"

# Import timezone data from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Import CA certificates from builder for HTTPS support
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Import the passwd file for the non-root user
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy the compiled binary from the builder stage
COPY --from=builder /app/compliance-audit-service /usr/local/bin/compliance-audit-service

# Use non-root user for security
USER appuser:appuser

# Expose the application port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/usr/local/bin/compliance-audit-service", "-health-check"]

# Set the entrypoint to the compiled binary
ENTRYPOINT ["/usr/local/bin/compliance-audit-service"]
