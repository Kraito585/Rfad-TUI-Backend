# ==========================================
# Этап 1: Сборка (Builder)
# ==========================================
FROM golang:1.25-alpine AS builder

# Устанавливаем зависимости ОС
RUN apk add --no-cache git

WORKDIR /app

# Кэшируем модули
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем бинарник (оставил имя auth-service, как было у вас)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/auth-service ./cmd/api

# ==========================================
# Этап 2: Финальный легковесный образ (Runtime)
# ==========================================
FROM alpine:latest

WORKDIR /app

# Добавляем сертификаты, tzdata и openssl (для генерации JWT ключей)
RUN apk --no-cache add ca-certificates tzdata openssl

# Копируем бинарник, миграции и конфиг
COPY --from=builder /bin/auth-service ./auth-service
COPY migrations/ ./migrations/
COPY config.yml ./config.yml
COPY credentials.json ./credentials.json 

# Создаем скрипт инициализации (Entrypoint) прямо внутри Dockerfile
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

# Обновленный порт из вашего config.yml
EXPOSE 7910

# Указываем скрипт как точку входа, а сам бинарник как аргумент
ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["./auth-service"]