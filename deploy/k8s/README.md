# Kubernetes-манифесты мессенджера

Полная документация по развёртыванию Go-бэкенда мессенджера в Kubernetes (minikube).

Сервисы:
- [`auth_service/`](../../auth_service) — аутентификация и авторизация
- [`main_app/`](../../main_app) — основная бизнес-логика
- [`websocket_service/`](../../websocket_service) — WebSocket-соединения

---

## Содержание

1. [Структура манифестов](#структура-манифестов)
2. [Переменные окружения](#переменные-окружения)
3. [Быстрый старт](#быстрый-старт)
4. [Команды для работы с кластером](#команды-для-работы-с-кластером)
5. [Архитектурные решения](#архитектурные-решения)
6. [Troubleshooting](#troubleshooting)

---

## Структура манифестов

```
deploy/k8s/
├── namespace.yaml          # Namespace "messenger"
├── configmap.yaml          # Конфигурация (порты, хосты, адреса)
├── secret.example.yaml     # Шаблон секретов (коммитится в git)
├── secret.yaml             # Реальные секреты (НЕ коммитится в git)
├── infrastructure.yaml     # PostgreSQL + RabbitMQ + MongoDB (StatefulSet)
├── backup-postgres.yaml    # CronJob для бэкапов PostgreSQL
├── backup-mongodb.yaml     # CronJob для бэкапов MongoDB
├── auth-service.yaml       # Deployment + Service для auth-service
├── main-app.yaml           # Deployment + Service для main-app
├── websocket-service.yaml  # Deployment + Service для websocket-service
├── ingress.yaml            # HTTP/WebSocket маршрутизация
└── deploy.sh               # Скрипт автоматического деплоя
```

### [`namespace.yaml`](./namespace.yaml)

Создаёт namespace `messenger` для изоляции всех ресурсов кластера.

```yaml
kind: Namespace
metadata:
  name: messenger
```

Все последующие ресурсы создаются в этом namespace.

---

### [`configmap.yaml`](./configmap.yaml)

Хранит не-секретную конфигурацию, которая передаётся в контейнеры через `envFrom`.

| Ключ | Значение | Описание |
|---|---|---|
| `APP_ENV` | `dev` | Окружение приложения |
| `MAIN_APP_HTTP_PORT` | `8080` | HTTP-порт main-app |
| `MAIN_APP_GRPC_PORT` | `8082` | gRPC-порт main-app |
| `AUTH_SERVICE_GRPC_PORT` | `8081` | gRPC-порт auth-service |
| `AUTH_SERVICE_METRICS_PORT` | `8087` | Порт метрик auth-service |
| `WEBSOCKET_SERVICE_HTTP_PORT` | `8083` | HTTP-порт websocket-service |
| `POSTGRES_HOST` | `postgres` | Хост PostgreSQL (имя Service) |
| `POSTGRES_PORT` | `5432` | Порт PostgreSQL |
| `POSTGRES_DB` | `patefon` | Имя базы данных |
| `POSTGRES_SSLMODE` | `disable` | SSL-режим |
| `POSTGRES_POOL_MAX_CONNS` | `10` | Максимум соединений в пуле |
| `RABBITMQ_HOST` | `rabbitmq` | Хост RabbitMQ (имя Service) |
| `RABBITMQ_PORT` | `5672` | AMQP-порт RabbitMQ |
| `RABBITMQ_VHOST` | `/` | Virtual host RabbitMQ |
| `MONGODB_HOST` | `mongodb` | Хост MongoDB (имя Service) |
| `MONGODB_PORT` | `27017` | Порт MongoDB |
| `MONGODB_DATABASE` | `files` | Имя базы данных MongoDB |
| `MONGODB_AUTH_SOURCE` | `files` | База аутентификации MongoDB |
| `AUTH_SERVICE_HOST` | `auth-service` | Хост auth-service (имя Service) |
| `AUTH_SERVICE_PORT` | `8081` | Порт auth-service для gRPC |
| `UPLOADS_PATH` | `/uploads` | Путь к директории загрузок |

---

### [`secret.example.yaml`](./secret.example.yaml) / `secret.yaml`

Хранит чувствительные данные (credentials). В git коммитится только шаблон `secret.example.yaml`.

| Ключ | Описание |
|---|---|
| `POSTGRES_USER` | Пользователь PostgreSQL |
| `POSTGRES_PASSWORD` | Пароль PostgreSQL |
| `RABBITMQ_USER` | Пользователь RabbitMQ |
| `RABBITMQ_PASSWORD` | Пароль RabbitMQ |
| `MONGODB_USER` | Пользователь MongoDB |
| `MONGODB_PASSWORD` | Пароль MongoDB |
| `JWT_SECRET` | Секрет для подписи JWT-токенов |
| `SESSION_SECRET` | Секрет для сессий |

Создание реального файла секретов:

```bash
cp deploy/k8s/secret.example.yaml deploy/k8s/secret.yaml
# Отредактировать secret.yaml — заменить все 'change-me' на реальные значения
```

> `secret.yaml` добавлен в `.gitignore` и не должен попадать в репозиторий.

---

### [`infrastructure.yaml`](./infrastructure.yaml)

Содержит StatefulSet + Service для трёх инфраструктурных компонентов:

#### PostgreSQL (`postgres:15-alpine`)
- **Тип:** StatefulSet с PersistentVolumeClaim (10Gi)
- Порт: `5432`
- Credentials берутся из Secret (`POSTGRES_USER`, `POSTGRES_PASSWORD`)
- База данных берётся из ConfigMap (`POSTGRES_DB`)
- readinessProbe: `pg_isready`
- **Хранение данных:** PersistentVolumeClaim `postgres-storage` (10Gi)
- **Headless Service:** `postgres-headless` для StatefulSet

#### RabbitMQ (`rabbitmq:3-management-alpine`)
- **Тип:** Deployment (без PVC)
- AMQP-порт: `5672`
- Management UI: `15672`
- Credentials берутся из Secret (`RABBITMQ_USER`, `RABBITMQ_PASSWORD`)
- readinessProbe: `tcpSocket` на порт 5672

#### MongoDB (`mongo:6`)
- **Тип:** StatefulSet с PersistentVolumeClaim (10Gi)
- Порт: `27017`
- Инициализационный скрипт монтируется из ConfigMap `mongodb-init-script`
- Скрипт создаёт пользователя `user` с доступом к базе `files`
- readinessProbe: `mongosh --eval "db.adminCommand('ping')"`
- **Хранение данных:** PersistentVolumeClaim `mongodb-storage` (10Gi)
- **Headless Service:** `mongodb-headless` для StatefulSet

> **Важно:** PostgreSQL и MongoDB используют StatefulSet с PersistentVolumeClaim —
> данные сохраняются при перезапуске подов. RabbitMQ остаётся без PVC (ephemeral).

---

### [`auth-service.yaml`](./auth-service.yaml)

Deployment + Service для сервиса аутентификации.

| Параметр | Значение |
|---|---|
| Образ | `messenger/auth-service:latest` |
| `imagePullPolicy` | `Never` (образ собирается локально в minikube) |
| Реплики | 2 |
| gRPC-порт | 8081 |
| Метрики-порт | 8087 |
| CPU request/limit | 100m / 500m |
| Memory request/limit | 128Mi / 512Mi |
| Probes | `tcpSocket` на порт 8081 |

---

### [`main-app.yaml`](./main-app.yaml)

Deployment + Service для основного приложения.

| Параметр | Значение |
|---|---|
| Образ | `messenger/main-app:latest` |
| `imagePullPolicy` | `Never` (образ собирается локально в minikube) |
| Реплики | 2 |
| HTTP-порт | 8080 |
| gRPC-порт | 8082 |
| CPU request/limit | 100m / 500m |
| Memory request/limit | 128Mi / 512Mi |
| Probes | `tcpSocket` на порт 8080 |

> **Примечание:** `main_app` собирается с `CGO_ENABLED=1` из-за зависимости
> `github.com/chai2010/webp`. Dockerfile использует multi-stage сборку с gcc.

---

### [`websocket-service.yaml`](./websocket-service.yaml)

Deployment + Service для WebSocket-сервиса.

| Параметр | Значение |
|---|---|
| Образ | `messenger/websocket-service:latest` |
| `imagePullPolicy` | `Never` (образ собирается локально в minikube) |
| Реплики | 2 |
| HTTP/WS-порт | 8083 |
| CPU request/limit | 100m / 500m |
| Memory request/limit | 128Mi / 512Mi |
| Probes | `tcpSocket` на порт 8083 |

**initContainer `wait-for-rabbitmq`:**

Перед запуском основного контейнера выполняется проверка доступности RabbitMQ:

```yaml
initContainers:
  - name: wait-for-rabbitmq
    image: busybox
    command: ['sh', '-c', 'until nc -z rabbitmq 5672; do echo waiting for rabbitmq; sleep 5; done']
```

Под не перейдёт в состояние `Running`, пока RabbitMQ не ответит на TCP-соединение.

---

### [`backup-postgres.yaml`](./backup-postgres.yaml)

CronJob для автоматического резервного копирования PostgreSQL.

| Параметр | Значение |
|---|---|
| Расписание | Ежедневно в 02:00 UTC |
| Хранение бэкапов | PersistentVolumeClaim `postgres-backup-pvc` (20Gi) |
| Формат | SQL-дамп, сжатый gzip |
| Удаление старых бэкапов | Автоматически через 7 дней |

**Команда бэкапа:**
```bash
pg_dump -h postgres -U $POSTGRES_USER -d $POSTGRES_DB \
  --clean --if-exists --format=plain --no-owner --no-acl | gzip
```

**Применение:**
```bash
kubectl apply -f deploy/k8s/backup-postgres.yaml
```

**Восстановление из бэкапа:**
```bash
# Получить список бэкапов
kubectl exec -n messenger postgres-backup-<job-id> -- ls -lh /backup/

# Скопировать бэкап локально
kubectl cp -n messenger postgres-backup-<job-id>:/backup/postgres-backup-YYYYMMDD-HHMMSS.sql.gz ./backup.sql.gz

# Восстановить
gunzip -c backup.sql.gz | kubectl exec -i -n messenger statefulset/postgres -- psql -U $POSTGRES_USER -d $POSTGRES_DB
```

---

### [`backup-mongodb.yaml`](./backup-mongodb.yaml)

CronJob для автоматического резервного копирования MongoDB.

| Параметр | Значение |
|---|---|
| Расписание | Ежедневно в 03:00 UTC |
| Хранение бэкапов | PersistentVolumeClaim `mongodb-backup-pvc` (20Gi) |
| Формат | MongoDB archive, сжатый gzip |
| Удаление старых бэкапов | Автоматически через 7 дней |

**Команда бэкапа:**
```bash
mongodump --host mongodb --port 27017 --username root \
  --password $MONGODB_PASSWORD --authenticationDatabase=admin \
  --db files --archive=/backup/mongodb-backup-YYYYMMDD-HHMMSS.gz --gzip
```

**Применение:**
```bash
kubectl apply -f deploy/k8s/backup-mongodb.yaml
```

**Восстановление из бэкапа:**
```bash
# Получить список бэкапов
kubectl exec -n messenger mongodb-backup-<job-id> -- ls -lh /backup/

# Скопировать бэкап локально
kubectl cp -n messenger mongodb-backup-<job-id>:/backup/mongodb-backup-YYYYMMDD-HHMMSS.gz ./backup.gz

# Восстановить
kubectl cp -n messenger ./backup.gz mongodb-0:/tmp/backup.gz
kubectl exec -n messenger statefulset/mongodb -- mongorestore --archive=/tmp/backup.gz --gzip
```

---

### [`ingress.yaml`](./ingress.yaml)

HTTP/WebSocket маршрутизация через nginx ingress controller.

Маршруты:
- `/api/` → `main-app:8080`
- `/api/startwebsocket` → `websocket-service:8083` (с поддержкой WebSocket upgrade)

---

## Переменные окружения

Все переменные передаются в контейнеры через `envFrom`:

```yaml
envFrom:
  - configMapRef:
      name: messenger-config
  - secretRef:
      name: messenger-secrets
```

Итоговый набор переменных в каждом контейнере — объединение ConfigMap и Secret.

---

## Быстрый старт

### Через скрипт (рекомендуется)

```bash
# Из корня репозитория:
cp deploy/k8s/secret.example.yaml deploy/k8s/secret.yaml
# Отредактировать secret.yaml
./deploy/k8s/deploy.sh
```

### Вручную

```bash
# 1. Запустить minikube
minikube start --driver=docker

# 2. Включить ingress
minikube addons enable ingress

# 3. Настроить docker-env (ОБЯЗАТЕЛЬНО перед сборкой образов)
eval $(minikube docker-env)

# 4. Собрать образы
docker build -t messenger/auth-service:latest    -f auth_service/Dockerfile    .
docker build -t messenger/main-app:latest        -f main_app/Dockerfile        .
docker build -t messenger/websocket-service:latest -f websocket_service/Dockerfile .

# 5. Применить манифесты
kubectl apply -f deploy/k8s/namespace.yaml
kubectl apply -f deploy/k8s/configmap.yaml
kubectl apply -f deploy/k8s/secret.yaml
kubectl apply -f deploy/k8s/infrastructure.yaml
kubectl apply -f deploy/k8s/backup-postgres.yaml
kubectl apply -f deploy/k8s/backup-mongodb.yaml
kubectl apply -f deploy/k8s/auth-service.yaml
kubectl apply -f deploy/k8s/main-app.yaml
kubectl apply -f deploy/k8s/websocket-service.yaml
kubectl apply -f deploy/k8s/ingress.yaml

# 6. Проверить статус
kubectl get pods -n messenger
```

---

## Команды для работы с кластером

### Просмотр состояния

```bash
# Все поды
kubectl get pods -n messenger

# Все ресурсы
kubectl get all -n messenger

# Ingress
kubectl get ingress -n messenger

# Подробная информация о поде
kubectl describe pod <pod-name> -n messenger
```

### Логи

```bash
# Следить за логами в реальном времени
kubectl logs -n messenger deployment/main-app          -f
kubectl logs -n messenger deployment/auth-service      -f
kubectl logs -n messenger deployment/websocket-service -f

# Логи конкретного пода
kubectl logs -n messenger <pod-name>

# Логи предыдущего (упавшего) контейнера
kubectl logs -n messenger <pod-name> --previous

# Через Makefile
make k8s-logs-main
make k8s-logs-auth
make k8s-logs-ws
```

### Пересборка и перезапуск

```bash
# Пересобрать и перезапустить auth-service
eval $(minikube docker-env)
docker build -t messenger/auth-service:latest -f auth_service/Dockerfile .
kubectl rollout restart deployment/auth-service -n messenger
kubectl rollout status  deployment/auth-service -n messenger

# Пересобрать и перезапустить main-app
eval $(minikube docker-env)
docker build -t messenger/main-app:latest -f main_app/Dockerfile .
kubectl rollout restart deployment/main-app -n messenger
kubectl rollout status  deployment/main-app -n messenger

# Пересобрать и перезапустить websocket-service
eval $(minikube docker-env)
docker build -t messenger/websocket-service:latest -f websocket_service/Dockerfile .
kubectl rollout restart deployment/websocket-service -n messenger
kubectl rollout status  deployment/websocket-service -n messenger

# Через Makefile (рекомендуется)
make k8s-rebuild-auth   # auth-service
make k8s-rebuild-main   # main-app
make k8s-rebuild-ws     # websocket-service
make k8s-rebuild-all    # все три сервиса
```

### Проброс портов для тестирования

```bash
kubectl port-forward -n messenger deployment/main-app          8080:8080 &
kubectl port-forward -n messenger deployment/auth-service      8081:8081 &
kubectl port-forward -n messenger deployment/websocket-service 8083:8083 &
```

После этого:
- Swagger UI: http://localhost:8080/docs/
- auth-service gRPC: `localhost:8081`
- websocket-service: `http://localhost:8083`

### Проверка WebSocket

```bash
curl -v --include --no-buffer \
  -H "Connection: Upgrade" \
  -H "Upgrade: websocket" \
  -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
  -H "Sec-WebSocket-Version: 13" \
  http://localhost:8083/api/startwebsocket
```

### Остановка и удаление

```bash
# Удалить все ресурсы namespace messenger
kubectl delete namespace messenger

# Остановить minikube (сохраняет состояние)
minikube stop

# Полностью удалить кластер
minikube delete
```

---

## Архитектурные решения

### `imagePullPolicy: Never`

Все три сервиса используют `imagePullPolicy: Never`. Это означает, что Kubernetes
не пытается скачать образ из registry, а использует только локально доступные образы.

Образы должны быть собраны **внутри** minikube-окружения:

```bash
eval $(minikube docker-env)   # переключить docker CLI на minikube
docker build ...              # образ попадает в minikube, а не в локальный Docker
```

### Self-healing

Автоматическое восстановление обеспечивается:

- **`replicas: 2`** — при падении одного пода трафик идёт на второй
- **`restartPolicy: Always`** (по умолчанию в Deployment) — упавший контейнер перезапускается
- **`startupProbe`** — даёт время на инициализацию (30 × 5с = 150с)
- **`readinessProbe`** — под не получает трафик, пока не готов
- **`livenessProbe`** — перезапускает зависший контейнер
- **`terminationGracePeriodSeconds: 30`** — graceful shutdown

### initContainer в websocket-service

`websocket-service` зависит от RabbitMQ. initContainer `wait-for-rabbitmq`
блокирует запуск основного контейнера до тех пор, пока RabbitMQ не ответит
на TCP-соединение на порту 5672.

### Почему нет PVC

PVC для [`uploads/`](../../uploads) не добавлен намеренно. В коде
[`main_app/cmd/app/main.go`](../../main_app/cmd/app/main.go) есть пометка
`TODO удалить uploads`, а основная файловая подсистема работает через MongoDB GridFS.
Директория `uploads/` используется как legacy/static-слой. Для stateless-развёртывания
это не оформлено как обязательное постоянное хранилище.

---

## Troubleshooting

### `ErrImagePull` / `ImagePullBackOff`

**Симптом:** под не стартует, статус `ErrImagePull` или `ImagePullBackOff`.

**Причина:** образ был собран в локальном Docker, а не внутри minikube.

**Решение:**

```bash
# Переключиться на docker-env minikube
eval $(minikube docker-env)

# Пересобрать образ
docker build -t messenger/<service-name>:latest -f <service>/Dockerfile .

# Перезапустить деплоймент
kubectl rollout restart deployment/<service-name> -n messenger
```

---

### `CrashLoopBackOff`

**Симптом:** под постоянно перезапускается, статус `CrashLoopBackOff`.

**Причина:** приложение падает при старте (ошибка конфигурации, недоступна БД и т.д.).

**Решение:**

```bash
# Посмотреть логи текущего контейнера
kubectl logs -n messenger <pod-name>

# Посмотреть логи предыдущего (упавшего) контейнера
kubectl logs -n messenger <pod-name> --previous

# Подробная информация о поде (события, причина перезапуска)
kubectl describe pod <pod-name> -n messenger
```

Частые причины:
- Неверные credentials в `secret.yaml`
- PostgreSQL/RabbitMQ/MongoDB ещё не готовы (подождать)
- Ошибка в конфигурации ConfigMap

---

### Под не стартует, ждёт initContainer

**Симптом:** под в статусе `Init:0/1` долгое время.

**Причина:** `websocket-service` ждёт, пока RabbitMQ станет доступен.

**Решение:** подождать, пока RabbitMQ перейдёт в статус `Running`:

```bash
# Проверить статус RabbitMQ
kubectl get pods -n messenger -l app=rabbitmq

# Посмотреть логи initContainer
kubectl logs -n messenger <websocket-pod-name> -c wait-for-rabbitmq
```

RabbitMQ может стартовать до 2 минут. После его готовности initContainer завершится
и основной контейнер запустится автоматически.

---

### `connection refused` при port-forward

**Симптом:** `curl: (7) Failed to connect to localhost port 8080: Connection refused`

**Причина:** под ещё не готов или port-forward не запущен.

**Решение:**

```bash
# Проверить статус подов
kubectl get pods -n messenger

# Убедиться, что под в статусе Running и READY 1/1
# Затем запустить port-forward
kubectl port-forward -n messenger deployment/main-app 8080:8080
```

---

### Ingress не работает

**Симптом:** запросы через ingress не доходят до сервисов.

**Решение:**

```bash
# Проверить, включён ли ingress addon
minikube addons list | grep ingress

# Включить если нет
minikube addons enable ingress

# Проверить статус ingress-controller
kubectl get pods -n ingress-nginx

# Проверить ingress-ресурс
kubectl describe ingress -n messenger
```

---

### Данные в БД пропали после перезапуска

**Причина:** RabbitMQ работает без PersistentVolumeClaim — данные хранятся в ephemeral-хранилище пода.

**Решение для разработки:** применить миграции заново после перезапуска:

```bash
# Применить SQL-миграции к PostgreSQL
kubectl exec -n messenger statefulset/postgres -- \
  psql -U <user> -d patefon -f /path/to/migration.sql
```

> **Важно:** PostgreSQL и MongoDB используют StatefulSet с PersistentVolumeClaim —
> данные сохраняются при перезапуске подов. RabbitMQ остаётся без PVC (ephemeral).

---

### Полный сброс и повторный деплой

```bash
# Удалить всё в namespace messenger
kubectl delete namespace messenger

# Подождать удаления
kubectl wait --for=delete namespace/messenger --timeout=60s

# Запустить деплой заново
./deploy/k8s/deploy.sh
```
