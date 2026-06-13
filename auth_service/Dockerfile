# Stage 1: builder
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Копируем go.mod и go.sum из корня репозитория и скачиваем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь исходный код
COPY . .

# Компилируем бинарник auth_service
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./auth_service/cmd/app/main.go

# Stage 2: runner
FROM alpine:latest AS runner

WORKDIR /app

# Копируем скомпилированный бинарник из builder
COPY --from=builder /app/server .

# Копируем файл конфига базы данных
COPY --from=builder /build/db/config/config.yaml ./db/config/config.yaml

EXPOSE 8081 8087

ENTRYPOINT ["/app/server"]
