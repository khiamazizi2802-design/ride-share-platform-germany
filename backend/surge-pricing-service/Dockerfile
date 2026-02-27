FROM golang:1.21-alpine AS builder

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -g 1000 appuser && adduser -D -u 1000 -G appuser appuser

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-w -s' -o main .

FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -g 1000 appuser && adduser -D -u 1000 -G appuser appuser

WORKDIR /app

COPY --from=builder --chown=appuser:appuser /app/main .

EXPOSE 8080

USER appuser

CMD ["./main"]
