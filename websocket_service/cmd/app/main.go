package main

import (
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/cors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/qwerty268/messenger_backend/protos/gen/go/authv1"
	authDelivery "github.com/qwerty268/messenger_backend/websocket_service/internal/middleware"
	"github.com/qwerty268/messenger_backend/websocket_service/internal/websocket/delivery"
	"github.com/qwerty268/messenger_backend/websocket_service/internal/websocket/usecase"
)

const (
	// host — DNS-имя K8s Service для gRPC чат-сервера main-app
	host = "main-app"
	port = 8082
)

func main() {
	// подключаем rebbit mq
	conn, err := amqp.Dial("amqp://root:root@rabbitmq:5672/") // Создаем подключение к RabbitMQ
	if err != nil {
		log.Fatalf("unable to open connect to RabbitMQ server. Error: %s", err)
	}
	defer func() {
		_ = conn.Close() // Закрываем подключение в случае удачной попытки
	}()

	log.Println("rebbit mq подключен")

	// Передаем соединение, а не один канал: внутри usecase будут созданы
	// отдельные AMQP-каналы для каждого consumer'а (RabbitMQ не позволяет
	// иметь больше одного consumer'а на канале).
	socketUsecase := usecase.NewWebsocketUsecase(conn, host, port)
	socketDelivery := delivery.NewWebsocket(*socketUsecase)

	router := mux.NewRouter()

	router = router.PathPrefix("/api/").Subrouter()

	// auth

	grpcConnAuth, err := grpc.Dial(
		"auth-service:8081",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer grpcConnAuth.Close()
	authClient := authv1.NewAuthClient(grpcConnAuth)

	auth := authDelivery.New(authClient)

	// ручки
	// index.html — опциональный (используется только для отладочной страницы /index)
	if _, err := os.Stat("index.html"); err == nil {
		tmpl := template.Must(template.ParseFiles("index.html"))
		router.HandleFunc("/index", func(w http.ResponseWriter, r *http.Request) {
			_ = tmpl.Execute(w, nil)
		})
	} else {
		log.Printf("index.html не найден (%v), пропускаем регистрацию /index", err)
	}

	router.HandleFunc("/startwebsocket", auth.Authorize(socketDelivery.HandleConnection))
	// мктрики
	router.Handle("/metrics", promhttp.Handler())

	c := cors.New(cors.Options{
		AllowOriginFunc:  func(origin string) bool { return true },
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "OPTIONS", "DELETE"},
		AllowedHeaders:   []string{"*"},
	})
	handler := c.Handler(router)

	log.Println("Starting server on :8083")
	if err := http.ListenAndServe(":8083", handler); err != nil {
		log.Fatal(err)
	}
}
