#!/usr/bin/env bash
# =============================================================================
# deploy.sh — скрипт полного деплоя мессенджера в minikube
#
# Запускать из корня репозитория:
#   ./deploy/k8s/deploy.sh
# =============================================================================

set -e

# -----------------------------------------------------------------------------
# Цвета для вывода
# -----------------------------------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# -----------------------------------------------------------------------------
# Вспомогательные функции вывода
# -----------------------------------------------------------------------------
info()    { echo -e "${BLUE}[INFO]${NC}  $*"; }
success() { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error()   { echo -e "${RED}[ERROR]${NC} $*" >&2; }
die()     { error "$*"; exit 1; }

# -----------------------------------------------------------------------------
# Константы
# -----------------------------------------------------------------------------
NAMESPACE="messenger"
K8S_DIR="deploy/k8s"
SECRET_FILE="${K8S_DIR}/secret.yaml"
SECRET_EXAMPLE="${K8S_DIR}/secret.example.yaml"

# -----------------------------------------------------------------------------
# Шаг 0: Проверка, что скрипт запущен из корня репозитория
# -----------------------------------------------------------------------------
if [[ ! -f "go.mod" ]]; then
  die "Скрипт должен запускаться из корня репозитория (там, где лежит go.mod)"
fi

echo ""
echo -e "${BLUE}╔══════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║        Деплой мессенджера в Kubernetes (minikube)        ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════╝${NC}"
echo ""

# -----------------------------------------------------------------------------
# Шаг 1: Проверка наличия необходимых инструментов
# -----------------------------------------------------------------------------
info "Проверка зависимостей..."

if ! command -v minikube &>/dev/null; then
  die "minikube не найден. Установите: brew install minikube"
fi
success "minikube найден: $(minikube version --short 2>/dev/null || minikube version | head -1)"

if ! command -v kubectl &>/dev/null; then
  die "kubectl не найден. Установите: brew install kubectl"
fi
success "kubectl найден: $(kubectl version --client --short 2>/dev/null || kubectl version --client | head -1)"

if ! command -v docker &>/dev/null; then
  die "docker не найден. Установите Docker Desktop: https://www.docker.com/products/docker-desktop"
fi

# -----------------------------------------------------------------------------
# Шаг 2: Проверка что Docker запущен
# -----------------------------------------------------------------------------
info "Проверка Docker..."
if ! docker info &>/dev/null; then
  die "Docker не запущен. Запустите Docker Desktop и повторите попытку."
fi
success "Docker запущен"

# -----------------------------------------------------------------------------
# Шаг 3: Проверка наличия secret.yaml
# -----------------------------------------------------------------------------
info "Проверка файла секретов..."
if [[ ! -f "${SECRET_FILE}" ]]; then
  warn "Файл ${SECRET_FILE} не найден!"
  echo ""
  echo "  Создайте его на основе примера:"
  echo ""
  echo -e "    ${YELLOW}cp ${SECRET_EXAMPLE} ${SECRET_FILE}${NC}"
  echo -e "    ${YELLOW}# Отредактируйте ${SECRET_FILE} — замените все 'change-me' на реальные значения${NC}"
  echo ""
  die "Деплой прерван: отсутствует ${SECRET_FILE}"
fi
success "Файл секретов найден: ${SECRET_FILE}"

# -----------------------------------------------------------------------------
# Шаг 4: Запуск minikube (если не запущен)
# -----------------------------------------------------------------------------
info "Проверка статуса minikube..."
MINIKUBE_STATUS=$(minikube status --format='{{.Host}}' 2>/dev/null || echo "Stopped")

if [[ "${MINIKUBE_STATUS}" != "Running" ]]; then
  info "Запуск minikube с драйвером docker..."
  minikube start --driver=docker
  success "minikube запущен"
else
  success "minikube уже запущен"
fi

# -----------------------------------------------------------------------------
# Шаг 5: Включение ingress addon
# -----------------------------------------------------------------------------
info "Включение ingress addon..."
minikube addons enable ingress
success "Ingress addon включён"

# -----------------------------------------------------------------------------
# Шаг 6: Настройка docker-env (сборка образов внутри minikube)
# -----------------------------------------------------------------------------
info "Настройка docker-env для minikube..."
eval "$(minikube docker-env)"
success "docker-env настроен — образы будут собираться внутри minikube"

# -----------------------------------------------------------------------------
# Шаг 7: Сборка Docker-образов
# -----------------------------------------------------------------------------
echo ""
info "Сборка Docker-образов..."

info "  [1/3] Сборка messenger/auth-service:latest..."
docker build -t messenger/auth-service:latest -f auth_service/Dockerfile .
success "  messenger/auth-service:latest собран"

info "  [2/3] Сборка messenger/main-app:latest..."
docker build -t messenger/main-app:latest -f main_app/Dockerfile .
success "  messenger/main-app:latest собран"

info "  [3/3] Сборка messenger/websocket-service:latest..."
docker build -t messenger/websocket-service:latest -f websocket_service/Dockerfile .
success "  messenger/websocket-service:latest собран"

echo ""
success "Все образы собраны"

# -----------------------------------------------------------------------------
# Шаг 8: Применение Kubernetes-манифестов в правильном порядке
# -----------------------------------------------------------------------------
echo ""
info "Применение Kubernetes-манифестов..."

MANIFESTS=(
  "${K8S_DIR}/namespace.yaml"
  "${K8S_DIR}/configmap.yaml"
  "${K8S_DIR}/secret.yaml"
  "${K8S_DIR}/infrastructure.yaml"
  "${K8S_DIR}/auth-service.yaml"
  "${K8S_DIR}/main-app.yaml"
  "${K8S_DIR}/websocket-service.yaml"
  "${K8S_DIR}/ingress.yaml"
)

for manifest in "${MANIFESTS[@]}"; do
  if [[ ! -f "${manifest}" ]]; then
    die "Манифест не найден: ${manifest}"
  fi
  info "  Применяю ${manifest}..."
  kubectl apply -f "${manifest}"
done

echo ""
success "Все манифесты применены"

# -----------------------------------------------------------------------------
# Шаг 9: Ожидание готовности инфраструктуры (PostgreSQL, RabbitMQ, MongoDB)
# -----------------------------------------------------------------------------
echo ""
info "Ожидание готовности инфраструктуры (PostgreSQL, RabbitMQ, MongoDB)..."
info "  Это может занять до 2 минут..."

kubectl wait --for=condition=ready pod \
  -l app=postgres \
  -n "${NAMESPACE}" \
  --timeout=120s 2>/dev/null && success "  PostgreSQL готов" || warn "  PostgreSQL ещё не готов, продолжаем..."

kubectl wait --for=condition=ready pod \
  -l app=rabbitmq \
  -n "${NAMESPACE}" \
  --timeout=120s 2>/dev/null && success "  RabbitMQ готов" || warn "  RabbitMQ ещё не готов, продолжаем..."

kubectl wait --for=condition=ready pod \
  -l app=mongodb \
  -n "${NAMESPACE}" \
  --timeout=120s 2>/dev/null && success "  MongoDB готова" || warn "  MongoDB ещё не готова, продолжаем..."

# -----------------------------------------------------------------------------
# Шаг 10: Ожидание готовности сервисов приложения
# -----------------------------------------------------------------------------
echo ""
info "Ожидание готовности сервисов приложения..."
info "  Это может занять до 3 минут (особенно websocket-service ждёт RabbitMQ)..."

kubectl wait --for=condition=ready pod \
  -l app.kubernetes.io/name=auth-service \
  -n "${NAMESPACE}" \
  --timeout=180s 2>/dev/null && success "  auth-service готов" || warn "  auth-service ещё не готов"

kubectl wait --for=condition=ready pod \
  -l app.kubernetes.io/name=main-app \
  -n "${NAMESPACE}" \
  --timeout=180s 2>/dev/null && success "  main-app готов" || warn "  main-app ещё не готов"

kubectl wait --for=condition=ready pod \
  -l app.kubernetes.io/name=websocket-service \
  -n "${NAMESPACE}" \
  --timeout=180s 2>/dev/null && success "  websocket-service готов" || warn "  websocket-service ещё не готов"

# -----------------------------------------------------------------------------
# Шаг 11: Финальный статус
# -----------------------------------------------------------------------------
echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║                   Деплой завершён!                      ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════════════╝${NC}"
echo ""

info "Статус подов в namespace ${NAMESPACE}:"
kubectl get pods -n "${NAMESPACE}"

echo ""
info "Статус сервисов:"
kubectl get services -n "${NAMESPACE}"

echo ""
echo -e "${YELLOW}╔══════════════════════════════════════════════════════════╗${NC}"
echo -e "${YELLOW}║              Как проверить работу сервисов               ║${NC}"
echo -e "${YELLOW}╚══════════════════════════════════════════════════════════╝${NC}"
echo ""
echo "  Пробросьте порты для локального тестирования:"
echo ""
echo -e "    ${GREEN}kubectl port-forward -n ${NAMESPACE} deployment/main-app 8080:8080 &${NC}"
echo -e "    ${GREEN}kubectl port-forward -n ${NAMESPACE} deployment/auth-service 8081:8081 &${NC}"
echo -e "    ${GREEN}kubectl port-forward -n ${NAMESPACE} deployment/websocket-service 8083:8083 &${NC}"
echo ""
echo "  Проверка main-app:"
echo -e "    ${GREEN}curl http://localhost:8080/docs/${NC}"
echo ""
echo "  Проверка WebSocket:"
echo -e "    ${GREEN}curl -v --include --no-buffer \\${NC}"
echo -e "    ${GREEN}  -H 'Connection: Upgrade' \\${NC}"
echo -e "    ${GREEN}  -H 'Upgrade: websocket' \\${NC}"
echo -e "    ${GREEN}  -H 'Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==' \\${NC}"
echo -e "    ${GREEN}  -H 'Sec-WebSocket-Version: 13' \\${NC}"
echo -e "    ${GREEN}  http://localhost:8083/api/startwebsocket${NC}"
echo ""
echo "  Логи сервисов:"
echo -e "    ${GREEN}kubectl logs -n ${NAMESPACE} deployment/main-app -f${NC}"
echo -e "    ${GREEN}kubectl logs -n ${NAMESPACE} deployment/auth-service -f${NC}"
echo -e "    ${GREEN}kubectl logs -n ${NAMESPACE} deployment/websocket-service -f${NC}"
echo ""
echo "  Пересборка отдельного сервиса (пример для websocket-service):"
echo -e "    ${GREEN}eval \$(minikube docker-env)${NC}"
echo -e "    ${GREEN}docker build -t messenger/websocket-service:latest ./websocket_service/${NC}"
echo -e "    ${GREEN}kubectl rollout restart deployment/websocket-service -n ${NAMESPACE}${NC}"
echo ""
echo "  Или используйте Makefile:"
echo -e "    ${GREEN}make k8s-rebuild-ws${NC}   # пересобрать websocket-service"
echo -e "    ${GREEN}make k8s-rebuild-all${NC}  # пересобрать все сервисы"
echo -e "    ${GREEN}make k8s-status${NC}       # статус подов"
echo ""
