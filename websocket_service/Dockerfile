# Stage 1: builder
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Копируем go.mod и go.sum из корня репозитория и скачиваем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь исходный код
COPY . .

# Компилируем бинарник websocket_service
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./websocket_service/cmd/app/main.go

# Stage 2: runner
FROM alpine:latest AS runner

WORKDIR /app

# Копируем только скомпилированный бинарник из builder
COPY --from=builder /app/server .
COPY index.html .

ENTRYPOINT ["/app/server"]
