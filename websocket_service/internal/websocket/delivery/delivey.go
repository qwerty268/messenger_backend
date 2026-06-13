package delivery

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"github.com/qwerty268/messenger_backend/global_utils/logger"
	"github.com/qwerty268/messenger_backend/global_utils/metric"
	"github.com/qwerty268/messenger_backend/global_utils/responser"
	"github.com/qwerty268/messenger_backend/websocket_service/internal/middleware"
	websocketUsecase "github.com/qwerty268/messenger_backend/websocket_service/internal/websocket/usecase"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 10048,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Webcosket struct {
	usecase websocketUsecase.WebsocketUsecase
}

func NewWebsocket(usecase websocketUsecase.WebsocketUsecase) Webcosket {
	return Webcosket{
		usecase: usecase,
	}
}

func (h *Webcosket) HandleConnection(w http.ResponseWriter, r *http.Request) {
	metric.IncHit()
	log := logger.LoggerWithCtx(r.Context(), logger.Log)
	user, ok := r.Context().Value(middleware.UserKey).(middleware.User)
	if !ok {
		responser.SendError(r.Context(), w, "Не переданы параметры", http.StatusInternalServerError)
		return
	}

	log.Printf("Пользователь %v Открыл сокет", user.ID)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Delivery: error during connection upgrade:", err)
		responser.SendError(r.Context(), w, "Delivery: error during connection upgrade:", http.StatusInternalServerError)
		return
	}
	defer log.Println("Message delivery: websocket is closing")
	defer conn.Close()

	eventChannel := make(chan websocketUsecase.AnyEvent, 10)

	err = h.usecase.InitBrokersForUser(user.ID, eventChannel)
	if err != nil {
		log.Errorf("Не удалось иницировать брокеры для пользователя")
		responser.SendError(r.Context(), w, "Нет нужных параметров", http.StatusInternalServerError)

		return
	}

	// пока соеденено
	duration := 500 * time.Millisecond

	for {
		select {
		case message := <-eventChannel:
			// запись новых сообщений
			log.Println("Message delivery websocket: получены новые сообщения")
			log.Println(message.Event)
			conn.WriteJSON(message.Event)

		default:
			time.Sleep(duration)
		}
	}
}

type ErrorResponse struct {
	Error  string `json:"error" example:"error message"`
	Status string `json:"status" example:"error"`
}
