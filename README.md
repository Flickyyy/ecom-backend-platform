# Go E-commerce API

Backend-платформа интернет-магазина. REST API для управления товарами, корзиной, заказами и пользователями.

## Стек технологий

- **Go 1.22** — основной язык
- **Gin** — HTTP-роутер
- **PostgreSQL 16** — основное хранилище (pgx/v5)
- **Redis 7** — кэширование каталога и ключи идемпотентности
- **RabbitMQ 3.13** — асинхронная обработка заказов (DLQ)
- **JWT** — аутентификация (golang-jwt)
- **slog** — структурированное логирование (stdlib)
- **Docker & Docker Compose** — контейнеризация
- **GitHub Actions** — CI/CD

## Архитектура

Clean Architecture с разделением на слои:

```
cmd/api/main.go              → точка входа, graceful shutdown
internal/
  config/config.go           → конфиг из env (caarlos0/env)
  model/model.go             → доменные модели
  dto/dto.go                 → request/response DTO
  repository/                → работа с БД (PostgreSQL)
  service/                   → бизнес-логика
  handler/                   → HTTP-хендлеры
  middleware/auth.go         → JWT middleware
  worker/order_worker.go     → RabbitMQ consumer
```

## Быстрый старт

```bash
docker-compose up --build
```

API доступен на `http://localhost:8080`.

## API Endpoints

### Auth
| Метод | Путь | Описание |
|---|---|---|
| POST | `/api/v1/auth/register` | Регистрация |
| POST | `/api/v1/auth/login` | Логин → JWT |

### Products
| Метод | Путь | Описание |
|---|---|---|
| GET | `/api/v1/products` | Список товаров (пагинация, поиск) |
| GET | `/api/v1/products/:id` | Один товар (кэш Redis) |
| POST | `/api/v1/products` | Создать товар (admin) |
| PUT | `/api/v1/products/:id` | Обновить товар (admin) |
| DELETE | `/api/v1/products/:id` | Удалить товар (admin) |

### Cart
| Метод | Путь | Описание |
|---|---|---|
| GET | `/api/v1/cart` | Получить корзину |
| POST | `/api/v1/cart/items` | Добавить товар |
| PUT | `/api/v1/cart/items/:id` | Изменить количество |
| DELETE | `/api/v1/cart/items/:id` | Удалить из корзины |

### Orders
| Метод | Путь | Описание |
|---|---|---|
| POST | `/api/v1/orders` | Создать заказ из корзины |
| GET | `/api/v1/orders` | Список заказов |
| GET | `/api/v1/orders/:id` | Детали заказа |

### Service
| Метод | Путь | Описание |
|---|---|---|
| GET | `/healthz` | Health check |
| GET | `/readyz` | Readiness check (PG + Redis + RMQ) |

## Обработка заказов

1. `POST /api/v1/orders` → статус `created`, сообщение в RabbitMQ
2. **Order Worker** потребляет сообщение:
   - Проверяет idempotency key в Redis
   - Транзакция: списание stock, обновление статуса → `completed`
   - При ошибке → `nack` → Dead Letter Queue
3. Idempotency key хранится 24h в Redis

## Тестирование

```bash
# Unit-тесты
go test ./internal/service/... -v

# Интеграционные тесты (нужен PostgreSQL)
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/ecommerce_test?sslmode=disable" \
  go test ./internal/repository/... -v

# Все тесты
go test -race ./...
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
| `REDIS_ADDR` | `localhost:6379` | Адрес Redis |
| `RABBITMQ_URL` | `amqp://guest:guest@localhost:5672/` | URL RabbitMQ |
| `JWT_SECRET` | `super-secret-key` | JWT секрет |
| `JWT_EXPIRATION` | `24h` | Время жизни токена |

## Лицензия

MIT
