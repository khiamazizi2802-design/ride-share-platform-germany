# Build stage
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY main.go .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o driver-onboarding-service .

# Final stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates

RUN adduser -D -g '' appuser

WORKDIR /home/appuser

COPY --from=builder /app/driver-onboarding-service .

EXPOSE 8080

USER appuser

CMD ["./driver-onboarding-service"]