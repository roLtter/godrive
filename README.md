# godrive

godrive — учебный проект облачного хранилища на Go.

Текущий стек:
- Go 1.25
- Gin (HTTP API)
- PostgreSQL (основная БД)
- Redis (кэш, сессии, rate limiting)
- MinIO (S3-совместимое объектное хранилище)
- golang-migrate (SQL-миграции)
- Zap (структурированные логи)

## Что реализовано

### Фаза 1 — Foundation
- Базовая структура проекта и точка входа сервера
- Инфраструктура через Docker Compose: PostgreSQL, Redis, MinIO, backend
- SQL-миграции (golang-migrate): таблицы `users`, `folders`, `files`, `shares`
- Конфиг через env-переменные (viper) с валидацией
- Структурированный логгер (zap) и middleware логирования HTTP-запросов
- MinIO клиент: инициализация bucket, presigned URL helpers
- Redis клиент: connection pool, health check

### Фаза 2 — Authentication
- `POST /api/auth/register` — регистрация, пароль хранится как bcrypt-хэш
- `POST /api/auth/login` — выдача JWT access token + refresh token
- `POST /api/auth/refresh` — ротация refresh token (старый инвалидируется в Redis)
- `POST /api/auth/logout` — удаление refresh token из Redis
- JWT middleware для защищённых роутов
- Rate limiting на auth-эндпоинтах (Redis sliding window, 10 req/min)
- Unit-тесты auth service, интеграционные тесты register/login/refresh flow

## Структура проекта

```
backend/
├── cmd/server/         — точка входа
├── internal/
│   ├── auth/           — регистрация, логин, JWT, middleware
│   ├── cache/redis/    — Redis клиент
│   ├── config/         — загрузка env-конфига
│   ├── logger/         — zap логгер
│   ├── middleware/     — HTTP middleware
│   ├── storage/minio/  — MinIO клиент
│   └── dbmigrate/      — pin migrate-драйверов
migrations/             — SQL миграции
docker-compose.yml
Makefile
```

## Требования

- Docker + Docker Compose
- Go 1.25+ (если запускать backend вне контейнера)

## Переменные окружения

Обязательные:
- `DB_URL` — строка подключения PostgreSQL
- `REDIS_URL` — строка подключения Redis
- `MINIO_URL` — адрес MinIO (например, `http://localhost:9000`)
- `MINIO_ROOT_USER` — логин MinIO
- `MINIO_ROOT_PASSWORD` — пароль MinIO
- `JWT_SECRET` — секрет для подписи JWT
- `JWT_ACCESS_TTL_MIN` — время жизни access token в минутах
- `JWT_REFRESH_TTL_DAYS` — время жизни refresh token в днях

Необязательные (с дефолтами):
- `PORT` (`8080`)
- `APP_ENV` (`dev`)
- `MIGRATIONS_PATH` (`migrations`)
- `MINIO_BUCKET` (`cloudstore`)
- `MINIO_PRESIGN_TTL_MIN` (`15`)
- `REDIS_POOL_SIZE` (`20`)
- `REDIS_MIN_IDLE_CONNS` (`5`)
- `REDIS_TIMEOUT_MS` (`5000`)
- `RATE_LIMIT_AUTH_RPM` (`10`)

## Локальный запуск

### Вариант 1: полностью в Docker (рекомендуется)

```bash
docker compose up --build
```

Проверить health:
```bash
curl http://localhost:8080/health
```

MinIO Console: `http://localhost:9001`

### Вариант 2: сервис локально, зависимости в Docker

```bash
docker compose up -d postgres redis minio
make migrate-up
make run
```

## API

### Auth

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/api/auth/register` | Регистрация |
| POST | `/api/auth/login` | Вход, получение токенов |
| POST | `/api/auth/refresh` | Обновление access token |
| POST | `/api/auth/logout` | Выход, инвалидация refresh token |

### System

| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/health` | Статус сервиса и зависимостей |

## Полезные команды

```bash
make run          # запуск backend
make build        # сборка проекта
make test         # все тесты
make migrate-up   # применить миграции
make migrate-down # откатить 1 миграцию
```

## Статус

| Фаза | Статус |
|------|--------|
| 1 — Foundation | Реализовано |
| 2 — Authentication | Реализовано |
| 3 — File Core | Запланировано |
| 4 — Sharing | Запланировано |
| 5 — Frontend | Запланировано |
| 6 — Deploy | Запланировано |
