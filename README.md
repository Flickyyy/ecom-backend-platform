# Go E-commerce API

Backend интернет-магазина на Go. REST API для товаров, корзины, заказов и авторизации.  
Асинхронная обработка заказов через RabbitMQ с Dead Letter Queue и идемпотентностью.

## Стек

- **Go 1.22**, Gin
- **PostgreSQL 16** (pgx/v5) — транзакции, индексы, миграции
- **RabbitMQ** — async order processing, manual ack, DLQ
- **Redis** — кэширование продуктов, идемпотентность
- **JWT** авторизация (customer / admin)
- **slog** — структурированное логирование (JSON)
- **Docker & Docker Compose**
- **GitHub Actions** CI (lint, test, build)

## Структура

```
cmd/api/main.go                → точка входа, graceful shutdown
internal/
  config/config.go             → конфиг из env
  model/model.go               → доменные модели
  dto/dto.go                   → request/response DTO
  repository/                  → слой данных (PostgreSQL)
  service/                     → бизнес-логика
  handler/                     → HTTP-хендлеры
  middleware/auth.go           → JWT middleware
  worker/order_worker.go       → RabbitMQ consumer (DLQ, idempotency)
migrations/                    → SQL миграции
```

## Запуск

```bash
docker-compose up --build
```

API: `http://localhost:8080`

## Endpoints

| Метод | Путь | Описание |
|---|---|---|
| POST | `/api/v1/auth/register` | Регистрация |
| POST | `/api/v1/auth/login` | Логин → JWT |
| GET | `/api/v1/products` | Список товаров |
| GET | `/api/v1/products/:id` | Товар по ID |
| POST | `/api/v1/products` | Создать (admin) |
| PUT | `/api/v1/products/:id` | Обновить (admin) |
| DELETE | `/api/v1/products/:id` | Удалить (admin) |
| GET | `/api/v1/cart` | Корзина |
| POST | `/api/v1/cart/items` | Добавить в корзину |
| PUT | `/api/v1/cart/items/:id` | Изменить количество |
| DELETE | `/api/v1/cart/items/:id` | Удалить из корзины |
| POST | `/api/v1/orders` | Создать заказ |
| GET | `/api/v1/orders` | Список заказов |
| GET | `/api/v1/orders/:id` | Детали заказа |
| GET | `/healthz` | Health check |
| GET | `/readyz` | Readiness (PG + Redis) |

## Тесты

```bash
# unit-тесты
go test -v ./internal/service/...

# интеграционные тесты (требуется PostgreSQL)
go test -v -tags integration ./internal/repository/...
```

## Переменные окружения

| Переменная | По умолчанию | Описание |
|---|---|---|
| `SERVER_PORT` | `8080` | Порт |
| `DB_HOST` | `localhost` | Хост PostgreSQL |
| `DB_PORT` | `5432` | Порт PostgreSQL |
| `DB_USER` | `postgres` | Пользователь |
| `DB_PASSWORD` | `postgres` | Пароль |
| `DB_NAME` | `ecommerce` | Имя БД |
| `JWT_SECRET` | `super-secret-key` | JWT секрет |
| `JWT_EXPIRATION` | `24h` | Время жизни токена |
| `REDIS_ADDR` | `localhost:6379` | Адрес Redis |
| `RABBITMQ_URL` | `amqp://guest:guest@localhost:5672/` | URL RabbitMQ |
