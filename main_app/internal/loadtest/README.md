# Load Test Handler

Тестовый endpoint для нагрузочного тестирования системы мессенджера.

## Endpoint

`POST /api/loadtest/message`

## Описание

Endpoint позволяет отправлять тестовые сообщения без CSRF защиты для нагрузочного тестирования.

## Формат запроса

```json
{
  "chat_id": "test-chat-id",
  "content": "Test message N",
  "timestamp": 1234567890.123
}
```

## Параметры

- `chat_id` (string, обязательный) — ID чата
- `content` (string, обязательный) — содержимое сообщения
- `timestamp` (float64, опциональный) — время отправки в секундах (Unix timestamp)

## Формат ответа

```json
{
  "success": true,
  "message": "Message queued for load testing",
  "timestamp": 1234567890.123
}
```

## Использование

```bash
curl -X POST http://localhost:8080/api/loadtest/message \
  -H "Content-Type: application/json" \
  -d '{
    "chat_id": "test-chat-1",
    "content": "Test message",
    "timestamp": 1234567890.123
  }'
```

## Примечания

- Endpoint не требует CSRF токена
- Endpoint не требует аутентификации (для упрощения нагрузочного тестирования)
- В production этот endpoint должен быть отключён или защищён
