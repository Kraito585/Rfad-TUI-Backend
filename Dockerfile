# Этап 1: Сборка
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/auth-service ./cmd/api

# Этап 2: Runtime
FROM alpine:latest
WORKDIR /app
RUN apk --no-cache add ca-certificates tzdata openssl

COPY --from=builder /bin/auth-service ./auth-service
COPY migrations/ ./migrations/

EXPOSE 7910
# Запускаем бинарник напрямую
CMD ["./auth-service"]