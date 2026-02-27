# Stage 1: Builder
FROM golang:1.21-alpine AS builder

# Install necessary build tools
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a \
    -o support-service \
    ./cmd/server/main.go

# Stage 2: Final minimal image
FROM scratch

# Copy CA certificates for HTTPS calls
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /app/support-service /support-service

# Copy migrations
COPY --from=builder /app/migrations /migrations

# Expose application port
EXPOSE 8080

# Set environment defaults
ENV APP_ENV=production
ENV SERVER_PORT=8080

# Run the binary
ENTRYPOINT ["/support-service"]
