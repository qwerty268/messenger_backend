#!/bin/bash
# Скрипт инициализации PostgreSQL в Kubernetes (minikube)
# Заливает dump.sql и user.sql в под postgres-0 в namespace messenger

set -e

NAMESPACE="messenger"
POD="postgres-0"
DB_USER="user_for_patefon"
DB_NAME="patefon"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

DUMP_FILE="$PROJECT_ROOT/db/migrations/dump.sql"
USER_FILE="$PROJECT_ROOT/db/migrations/user.sql"

echo "=== Инициализация БД в Kubernetes ==="
echo "Namespace: $NAMESPACE"
echo "Pod: $POD"
echo "DB User: $DB_USER"
echo "DB Name: $DB_NAME"
echo ""

# Проверяем, что под запущен
echo "1. Проверяем, что под $POD запущен..."
kubectl get pod "$POD" -n "$NAMESPACE" > /dev/null 2>&1 || {
    echo "ОШИБКА: Под $POD не найден в namespace $NAMESPACE"
    exit 1
}
echo "   ✅ Под найден"

# Копируем файлы в под
echo "2. Копируем dump.sql в под..."
kubectl cp "$DUMP_FILE" "$NAMESPACE/$POD:/tmp/dump.sql"
echo "   ✅ dump.sql скопирован"

echo "3. Копируем user.sql в под..."
kubectl cp "$USER_FILE" "$NAMESPACE/$POD:/tmp/user.sql"
echo "   ✅ user.sql скопирован"

# Заливаем дамп (заменяем OWNER TO postgres на OWNER TO user_for_patefon)
echo "4. Заливаем dump.sql (с заменой владельца таблиц)..."
kubectl exec "$POD" -n "$NAMESPACE" -- bash -c \
    "sed 's/OWNER TO postgres/OWNER TO $DB_USER/g' /tmp/dump.sql | psql -U $DB_USER -d $DB_NAME"
echo "   ✅ dump.sql залит"

# Проверяем, что таблицы созданы
echo "5. Проверяем наличие таблиц..."
kubectl exec "$POD" -n "$NAMESPACE" -- psql -U "$DB_USER" -d "$DB_NAME" -c \
    "SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename;"
echo ""

# Перезапускаем поды приложений
echo "6. Перезапускаем поды приложений..."
kubectl rollout restart deployment/auth-service -n "$NAMESPACE" 2>/dev/null || echo "   ⚠️  auth-service deployment не найден"
kubectl rollout restart deployment/main-app -n "$NAMESPACE" 2>/dev/null || echo "   ⚠️  main-app deployment не найден"
kubectl rollout restart deployment/websocket-service -n "$NAMESPACE" 2>/dev/null || echo "   ⚠️  websocket-service deployment не найден"
echo "   ✅ Поды перезапущены"

echo ""
echo "=== Готово! ==="
echo "Подождите ~30 секунд, пока поды перезапустятся."
echo "Проверить статус: kubectl get pods -n $NAMESPACE"
