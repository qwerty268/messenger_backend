# Kubernetes manifests

Структура Kubernetes-манифестов для stateless-сервисов [`main_app/`](../../main_app), [`auth_service/`](../../auth_service) и [`websocket_service/`](../../websocket_service) находится в [`deploy/k8s/`](.).

## Что входит

- namespace [`messenger`](./namespace.yaml)
- общий [`ConfigMap`](./configmap.yaml) с внешними адресами PostgreSQL, RabbitMQ и MongoDB
- пример [`Secret`](./secret.example.yaml) для учетных данных
- [`Deployment`](./main-app.yaml) и [`Service`](./main-app.yaml) для `main-app`
- [`Deployment`](./auth-service.yaml) и [`Service`](./auth-service.yaml) для `auth-service`
- [`Deployment`](./websocket-service.yaml) и [`Service`](./websocket-service.yaml) для `websocket-service`
- [`Ingress`](./ingress.yaml) для HTTP и WebSocket

## Почему здесь нет PVC

PVC для [`uploads/`](../../uploads) не добавлен намеренно. В текущем коде [`main_app/cmd/app/main.go`](../../main_app/cmd/app/main.go:148) есть пометка `TODO удалить uploads`, а основная файловая подсистема уже работает через MongoDB GridFS. Каталог [`uploads/`](../../uploads) используется как legacy/static-слой для выдачи файлов и стикеров, но для stateless-развертывания это не оформлено как обязательное постоянное хранилище в Kubernetes-манифестах. Если потребуется сохранить именно файловую директорию внутри кластера, PVC можно добавить отдельно после подтверждения требований к shared storage.

## Self-healing

Автоматическое восстановление контейнеров обеспечивается комбинацией:
- [`Deployment`](./main-app.yaml) / [`Deployment`](./auth-service.yaml) / [`Deployment`](./websocket-service.yaml) с `replicas: 2`
- стандартной политики перезапуска Pod'ов внутри Deployment (`restartPolicy: Always` по умолчанию)
- [`startupProbe`](./main-app.yaml), [`readinessProbe`](./main-app.yaml), [`livenessProbe`](./main-app.yaml) и аналогичных probe для остальных сервисов
- `terminationGracePeriodSeconds`
- `resources.requests` и `resources.limits`

Такой набор позволяет Kubernetes:
- автоматически перезапускать контейнер при падении
- не направлять трафик в неготовый Pod
- заменить неработающий экземпляр новым
- переживать падение отдельного Pod за счет нескольких реплик

## Порты сервисов

- `main-app`: HTTP `8080`, gRPC `8082`
- `auth-service`: gRPC `8081`, metrics `8087`
- `websocket-service`: HTTP/WebSocket `8083`

Так как явные HTTP health-check endpoints в коде не выделены, в Deployment используются `tcpSocket` probes.

## Подготовка секрета

Сначала создайте реальный secret на основе шаблона [`deploy/k8s/secret.example.yaml`](./secret.example.yaml):

```bash
cp deploy/k8s/secret.example.yaml deploy/k8s/secret.yaml
```

После этого замените значения в [`deploy/k8s/secret.yaml`](./secret.example.yaml) на реальные.

> В репозиторий добавлен только пример секрета. Реальный secret коммитить не нужно.

## Применение

Применить все манифесты:

```bash
kubectl apply -f deploy/k8s/
```

Проверить ресурсы:

```bash
kubectl get all -n messenger
kubectl get ingress -n messenger
```

## Что нужно настроить перед запуском

Перед применением манифестов необходимо:
- собрать и опубликовать образы `messenger/main-app`, `messenger/auth-service`, `messenger/websocket-service`
- заменить хосты внешних зависимостей в [`ConfigMap`](./configmap.yaml)
- создать реальный secret вместо шаблона
- при необходимости скорректировать ingress class и hostname
