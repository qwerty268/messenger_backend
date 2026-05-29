package loadtest

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	amqp "github.com/rabbitmq/amqp091-go"

	socketUsecase "github.com/qwerty268/messenger_backend/global_utils/events"
)

var rabbitMQChannel *amqp.Channel

// InitRabbitMQ инициализирует канал RabbitMQ для нагрузочного тестирования
func InitRabbitMQ(ch *amqp.Channel) {
	rabbitMQChannel = ch
}

// LoadTestMessage структура для нагрузочного тестирования
type LoadTestMessage struct {
	ChatID    string  `json:"chat_id"`
	Content   string  `json:"content"`
	Timestamp float64 `json:"timestamp"`
}

// LoadTestHandler обрабатывает запросы нагрузочного тестирования
func LoadTestHandler(w http.ResponseWriter, r *http.Request) {
	// Проверка метода
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсинг JSON
	var msg LoadTestMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Добавляем timestamp, если не указан
	if msg.Timestamp == 0 {
		msg.Timestamp = float64(time.Now().UnixNano()) / 1e9
	}

	// Отправляем сообщение в RabbitMQ
	if rabbitMQChannel != nil {
		// Создаём структуру сообщения для WebSocket
		// Генерируем UUID для chat_id если он не валидный
		chatUUID, err := uuid.Parse(msg.ChatID)
		if err != nil {
			// Если chat_id не валидный UUID, генерируем новый на основе строки
			chatUUID = uuid.NewSHA1(uuid.NameSpaceURL, []byte(msg.ChatID))
		}

		messageEvent := socketUsecase.MessageEvent{
			Action: "message",
			Message: socketUsecase.Message{
				MessageId: uuid.New(),
				ChatId:    chatUUID,
				Message:   msg.Content,
				SentAt:    time.Unix(int64(msg.Timestamp), 0),
				AuthorID:  uuid.MustParse("39a9aea0-d461-437d-b4eb-bf030a0efc80"), // user11
			},
		}

		// Сериализуем сообщение
		body, err := socketUsecase.SerializeMessageEvent(messageEvent)
		if err != nil {
			http.Error(w, "Failed to serialize message", http.StatusInternalServerError)
			return
		}

		// Публикуем в очередь "message"
		ctx := context.Background()
		err = rabbitMQChannel.PublishWithContext(ctx,
			"",        // exchange
			"message", // routing key (имя очереди)
			false,     // mandatory
			false,     // immediate
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(body),
			})
		if err != nil {
			http.Error(w, "Failed to publish message", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"message":   "Message queued for load testing",
		"timestamp": msg.Timestamp,
	})
}

// RegisterRoutes регистрирует маршруты нагрузочного тестирования
func RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/loadtest/message", LoadTestHandler).Methods("POST")
}
