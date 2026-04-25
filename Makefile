# Устанавливаем имя исполняемого файла
BINARY_NAME=backend_app

# Путь к исходному коду
SRC_DIR=src

CMD_DIR=cmd/app

up:
	sudo chown 999:999 ./log ; \
	sudo chmod 700 ./log ; \
	sudo docker-compose down && sudo docker-compose up --build

down:
	sudo docker-compose down

# Устанавливаем команду для сборки
build:
	go build -o $(BINARY_NAME) $(CMD_DIR)/main.go

# Команда для очистки скомпилированных бинарников
clean:
	rm -f $(BINARY_NAME)

# Команда для запуска приложения
run: build
	./$(BINARY_NAME)

# Команда для генерации сваггера
swagger:
	swag init --parseDependency --parseInternal -g main_app/cmd/app/main.go

# Команда для запуска тестов
test:
	go test ./...

# Команда для отчета покрытия тестами
cover:
	go test -json ./... -coverprofile coverprofile_.tmp -coverpkg=./... ; \
	cat coverprofile_.tmp | grep -v mocks.go | grep -v .pb.go | grep -v _grpc.go | grep -v _mock.go | grep -v main.go | grep -v docs.go > coverprofile.tmp ; \
	rm coverprofile_.tmp ; \
	go tool cover -html coverprofile.tmp ; \
	go tool cover -func coverprofile.tmp

# Команда для установки зависимостей
deps:
	go mod tidy

# Команда запуска линтера
lint:
	golangci-lint run

# Команда автофикса линтера
lintfix:
	golangci-lint run --fix

# Kubernetes / minikube

NAMESPACE=messenger

# Пересобрать все сервисы и перезапустить поды
k8s-rebuild-all: k8s-rebuild-auth k8s-rebuild-main k8s-rebuild-ws

# Пересобрать auth-service и перезапустить деплоймент
k8s-rebuild-auth:
	eval $$(minikube docker-env) && docker build -t messenger/auth-service:latest -f auth_service/Dockerfile .
	kubectl rollout restart deployment/auth-service -n $(NAMESPACE)
	kubectl rollout status deployment/auth-service -n $(NAMESPACE) --timeout=120s

# Пересобрать main-app и перезапустить деплоймент
k8s-rebuild-main:
	eval $$(minikube docker-env) && docker build -t messenger/main-app:latest -f main_app/Dockerfile .
	kubectl rollout restart deployment/main-app -n $(NAMESPACE)
	kubectl rollout status deployment/main-app -n $(NAMESPACE) --timeout=120s

# Пересобрать websocket-service и перезапустить деплоймент
k8s-rebuild-ws:
	eval $$(minikube docker-env) && docker build -t messenger/websocket-service:latest -f websocket_service/Dockerfile .
	kubectl rollout restart deployment/websocket-service -n $(NAMESPACE)
	kubectl rollout status deployment/websocket-service -n $(NAMESPACE) --timeout=120s

# Статус подов
k8s-status:
	kubectl get pods -n $(NAMESPACE)

# Логи сервисов (последние 100 строк)
k8s-logs-auth:
	kubectl logs -n $(NAMESPACE) deployment/auth-service --tail=100 -f

k8s-logs-main:
	kubectl logs -n $(NAMESPACE) deployment/main-app --tail=100 -f

k8s-logs-ws:
	kubectl logs -n $(NAMESPACE) deployment/websocket-service --tail=100 -f

# Основная команда по умолчанию
.PHONY: build lint clean run deps \
	k8s-rebuild-all k8s-rebuild-auth k8s-rebuild-main k8s-rebuild-ws \
	k8s-status k8s-logs-auth k8s-logs-main k8s-logs-ws