package usecase

import (
	"context"
	"net"
	"strconv"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	chatModels "github.com/qwerty268/messenger_backend/global_utils/events"
	"github.com/qwerty268/messenger_backend/global_utils/logger"
	grpcChat "github.com/qwerty268/messenger_backend/protos/gen/go/chat"
)

// ивент может быть либо изменение сущности чата, либо сообщение.
type AnyEvent struct {
	TypeOfEvent string
	Event       interface{}
}

type ChatInfo struct {
	events chan chatModels.Event
	users  map[uuid.UUID]struct{}
}

type WebsocketUsecase struct {
	// отдельный канал для consumer'а сообщений
	chMessages *amqp.Channel
	// отдельный канал для consumer'а чатов
	chChats *amqp.Channel
	// мапа с чатами и каналами для ивентов по чатам
	onlineChats map[uuid.UUID]ChatInfo
	// мапа с онлайн пользователями и
	onlineUsers    map[uuid.UUID]chan AnyEvent
	chatRepository grpcChat.ChatServiceClient
}

func NewWebsocketUsecase(conn *amqp.Connection, host string, port int) *WebsocketUsecase {
	log := logger.LoggerWithCtx(context.Background(), logger.Log)

	// Объявляем очереди на отдельном временном канале.
	declareCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open declare channel. Error: %s", err)
	}

	if _, err := declareCh.QueueDeclare("message", false, false, false, false, nil); err != nil {
		log.Fatalf("failed to declare 'message' queue. Error: %s", err)
	}
	log.Infof("queue 'message' declared")

	if _, err := declareCh.QueueDeclare("chat", false, false, false, false, nil); err != nil {
		log.Fatalf("failed to declare 'chat' queue. Error: %s", err)
	}
	log.Infof("queue 'chat' declared")

	if err := declareCh.Close(); err != nil {
		log.Warnf("failed to close declare channel: %s", err)
	}

	// Создаем отдельный канал для consumer'а сообщений.
	// RabbitMQ (AMQP 0-9-1) не позволяет регистрировать несколько consumer'ов
	// с пустым/одинаковым consumer-tag на одном канале, поэтому для каждого
	// consumer'а используется свой канал.
	chMessages, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open messages channel. Error: %s", err)
	}
	log.Infof("messages channel opened")

	// Создаем отдельный канал для consumer'а чатов
	chChats, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open chats channel. Error: %s", err)
	}
	log.Infof("chats channel opened")

	grpcAddress := net.JoinHostPort(host, strconv.Itoa(port))
	// Создаем клиент
	cc, err := grpc.NewClient(grpcAddress,
		// Используем insecure-коннект для тестов
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic("s")
	}

	// gRPC-клиент сервера Auth
	grpcClient := grpcChat.NewChatServiceClient(cc)

	// Подписываемся на события закрытия — это поможет понять, по какой причине
	// AMQP-канал/соединение закрывается извне.
	connCloseCh := make(chan *amqp.Error, 1)
	conn.NotifyClose(connCloseCh)
	go func() {
		err, ok := <-connCloseCh
		if !ok {
			log.Warnf("amqp connection close notifier exited")
			return
		}
		log.Errorf("amqp CONNECTION closed: %+v", err)
	}()

	msgCloseCh := make(chan *amqp.Error, 1)
	chMessages.NotifyClose(msgCloseCh)
	go func() {
		err, ok := <-msgCloseCh
		if !ok {
			log.Warnf("messages channel close notifier exited")
			return
		}
		log.Errorf("messages CHANNEL closed: %+v", err)
	}()

	chatCloseCh := make(chan *amqp.Error, 1)
	chChats.NotifyClose(chatCloseCh)
	go func() {
		err, ok := <-chatCloseCh
		if !ok {
			log.Warnf("chats channel close notifier exited")
			return
		}
		log.Errorf("chats CHANNEL closed: %+v", err)
	}()

	socket := &WebsocketUsecase{
		chMessages:     chMessages,
		chChats:        chChats,
		onlineChats:    map[uuid.UUID]ChatInfo{},
		onlineUsers:    map[uuid.UUID]chan AnyEvent{},
		chatRepository: grpcClient,
	}

	go socket.consumeMessages()
	go socket.consumeChats()

	return socket
}

func (w *WebsocketUsecase) InitBrokersForUser(userId uuid.UUID, eventChannel chan AnyEvent) error {
	log := logger.LoggerWithCtx(context.Background(), logger.Log)

	chats, err := w.chatRepository.GetUserChats(context.Background(), &grpcChat.UserChatsRequest{UserId: userId.String()})
	if err != nil {
		log.Errorf("Не удалось запросить чаты: %v", err)
		return err
	}

	w.onlineUsers[userId] = eventChannel
	log.Infof("Пользователь %v онлайн", userId)

	// Добавляем в брокеры пользователей
	for _, chatId := range chats.ChatIds {
		chatUUID, err := uuid.Parse(chatId)
		if err != nil {
			continue
		}

		if chatInfo, ok := w.onlineChats[chatUUID]; ok {
			chatInfo.events <- chatModels.Event{
				Action: AddWebcosketUser,
				Users:  []uuid.UUID{userId},
			}
			log.Infof("Пользователь %v добавлен в брокер для чата %v", userId, chatId)
		}
	}
	return nil
}
