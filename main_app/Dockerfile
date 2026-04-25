# Stage 1: builder
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Устанавливаем C-зависимости для CGO (libwebp требует нативных символов)
RUN apk add --no-cache gcc musl-dev libwebp-dev

# Копируем go.mod и go.sum из корня репозитория и скачиваем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь исходный код
COPY . .

# Компилируем бинарник main_app с CGO (требуется для github.com/chai2010/webp)
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/server ./main_app/cmd/app/main.go

# Stage 2: runner
FROM alpine:latest AS runner

WORKDIR /app

# Устанавливаем runtime-библиотеку libwebp
RUN apk add --no-cache libwebp

# Копируем скомпилированный бинарник из builder
COPY --from=builder /app/server .

# Копируем файл конфига базы данных
COPY --from=builder /build/db/config/config.yaml ./db/config/config.yaml

EXPOSE 8082 8080

ENTRYPOINT ["/app/server"]
