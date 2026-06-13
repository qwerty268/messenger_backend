# Stage 1: builder
# Используем образ с поддержкой CGO, так как зависимость github.com/chai2010/webp
# требует libwebp/gcc при сборке.
FROM golang:1.23-alpine AS builder

# Инструменты сборки для cgo + заголовки libwebp.
RUN apk add --no-cache build-base libwebp-dev

WORKDIR /build

# Слой со скачанными модулями кешируется до изменений в go.mod/go.sum.
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь исходный код
COPY . .

# Собираем бинарник main_app с CGO (требуется github.com/chai2010/webp).
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/server ./main_app/cmd/app/main.go

# Stage 2: runner
# alpine с рантайм-библиотекой libwebp для динамически связанного бинарника.
FROM alpine:latest AS runner

RUN apk add --no-cache libwebp ca-certificates

WORKDIR /app

# Копируем скомпилированный бинарник из builder
COPY --from=builder /app/server .

# Конфиг базы данных — main_app читает его относительным путём
# ./db/config/config.yaml, поэтому копируем со структурой каталогов.
COPY --from=builder /build/db/config/config.yaml ./db/config/config.yaml

# Дефолтные стикеры. main_app при старте инициализирует их из путей
# /uploads/stickers/{pack}/{id}.webp, поэтому кладём в /uploads (внутри контейнера).
# Если рабочий каталог /app, абсолютные пути в коде указывают на корень /.
COPY --from=builder /build/uploads/stickers /uploads/stickers

# Документируем порты: 8080 — HTTP, 8082 — gRPC
EXPOSE 8080 8082

ENTRYPOINT ["/app/server"]
