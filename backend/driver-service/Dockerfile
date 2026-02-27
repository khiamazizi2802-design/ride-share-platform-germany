# Stage 1: Builder
FROM golang:1.21-alpine AS builder

# Install git and other build dependencies if needed
RUN apk add --no-cache git

# Set working directory
WORKDIR /build

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s' \
    -o driver-service \
    ./cmd/main.go

# Stage 2: Final
FROM alpine:latest

# Install ca-certificates and tzdata
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home /home/appuser \
    --shell /sbin/nologin \
    --uid 1000 \
    appuser

# Set working directory
WORKDIR /home/appuser

# Copy the binary from builder stage
COPY --from=builder /build/driver-service .

# Change ownership to appuser
RUN chown -R appuser:appuser /home/appuser

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Run the binary
ENTRYPOINT ["./driver-service"]