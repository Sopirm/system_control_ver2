# 🚀 Быстрый запуск системы

## Development окружение

```bash
# Клонирование и переход в проект
git clone <repository_url>
cd system_control_ver2

# Быстрый запуск всей системы
docker-compose build
docker-compose up -d

# Проверка состояния
docker-compose ps
curl http://localhost:8080/health
```

**Доступные сервисы:**
- API Gateway: http://localhost:8080
- Service Users: http://localhost:8081  
- Service Orders: http://localhost:8082
- PostgreSQL: localhost:5432 (system_control_dev/postgres/postgres)

## Test окружение

```bash
# Очистка dev окружения
docker-compose down -v

# Запуск тестового окружения
docker-compose -f docker-compose.test.yml build
docker-compose -f docker-compose.test.yml up -d

# Проверка
curl http://localhost:18080/health
```

**Доступные сервисы:**
- API Gateway: http://localhost:18080
- Service Users: http://localhost:18081
- Service Orders: http://localhost:18082  
- PostgreSQL: localhost:5433 (system_control_test/postgres/postgres)

## Production окружение

```bash
# Настройка переменных окружения
export DB_HOST=your_production_db
export DB_NAME=your_db_name
export DB_USER=your_db_user
export DB_PASSWORD=your_secure_password
export JWT_SECRET_FROM_VAULT=your_jwt_secret
export REDIS_PASSWORD=your_redis_password

# Запуск production
docker-compose -f docker-compose.production.yml build --no-cache
docker-compose -f docker-compose.production.yml up -d

# Проверка
curl https://your-domain.com/health
```

## Полезные команды

```bash
# Просмотр логов
docker-compose logs -f [service_name]

# Остановка
docker-compose down

# Полная очистка
docker-compose down -v
docker system prune -f

# Запуск отдельного сервиса
docker-compose up -d postgres
docker-compose up -d service_users

# Подключение к контейнеру
docker-compose exec service_name /bin/sh
```

## Troubleshooting

**Порт занят:**
```bash
sudo ss -tulpn | grep :8080
sudo kill -9 PID
```

**Контейнер не запускается:**
```bash
docker-compose logs service_name
docker-compose build --no-cache service_name
```

**База данных недоступна:**
```bash
docker-compose exec postgres pg_isready -U postgres
```
