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

# Скрипт инициализации оставляем
RUN echo '#!/bin/sh' > /app/entrypoint.sh && \
    echo 'mkdir -p ./config/certs' >> /app/entrypoint.sh && \
    echo 'if [ ! -f "./config/certs/private.pem" ]; then' >> /app/entrypoint.sh && \
    echo '  echo "🔑 RSA ключи не найдены. Генерируем новые..."' >> /app/entrypoint.sh && \
    echo '  openssl genpkey -algorithm RSA -out ./config/certs/private.pem -pkeyopt rsa_keygen_bits:2048 2>/dev/null' >> /app/entrypoint.sh && \
    echo '  openssl rsa -pubout -in ./config/certs/private.pem -out ./config/certs/public.pem 2>/dev/null' >> /app/entrypoint.sh && \
    echo '  echo "✅ Ключи успешно сгенерированы."' >> /app/entrypoint.sh && \
    echo 'fi' >> /app/entrypoint.sh && \
    echo 'exec "$@"' >> /app/entrypoint.sh && \
    chmod +x /app/entrypoint.sh

EXPOSE 7910
ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["./auth-service"]