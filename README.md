# CloudStore

CloudStore - учебный backend-проект облачного хранилища на Go.

Текущий стек:
- Go 1.25
- Gin (HTTP API)
- PostgreSQL (основная БД)
- Redis (кэш/сессионный слой)
- MinIO (S3-совместимое объектное хранилище)
- golang-migrate (SQL-миграции)
- Zap (структурированные логи)

## Что уже реализовано

- Базовая структура backend-проекта и точка входа сервера.
- Инфраструктура через Docker Compose: PostgreSQL, Redis, MinIO, backend.
- Подключены SQL-миграции (`golang-migrate`) и стартовый набор миграций.
- Схема БД: `users`, `folders`, `files`, `shares`.
- Конфиг через env-переменные (`viper`) с валидацией.
- Структурированный логгер (`zap`) и middleware логирования HTTP-запросов.
- MinIO клиент: инициализация bucket + presigned URL helpers.
- Redis клиент: connection pool + health check.

## Структура проекта

- `backend/cmd/server` - запуск HTTP-сервера.
- `backend/internal/config` - загрузка и валидация конфигурации.
- `backend/internal/logger` - инициализация логгера.
- `backend/internal/middleware` - HTTP middleware.
- `backend/internal/storage/minio` - клиент MinIO.
- `backend/internal/cache/redis` - клиент Redis.
- `backend/internal/dbmigrate` - pin импортов migrate-драйверов.
- `migrations` - SQL миграции.
- `docker-compose.yml` - локальная инфраструктура.
- `Makefile` - команды разработки.

## Требования

- Docker + Docker Compose
- Go 1.25+ (если запускать backend вне контейнера)

## Переменные окружения

Ключевые переменные:
- `DB_URL` - строка подключения PostgreSQL
- `REDIS_URL` - строка подключения Redis
- `MINIO_URL` - адрес MinIO (например, `http://localhost:9000`)
- `MINIO_ROOT_USER` - логин MinIO
- `MINIO_ROOT_PASSWORD` - пароль MinIO

Необязательные (с дефолтами):
- `PORT` (`8080`)
- `APP_ENV` (`dev`)
- `MIGRATIONS_PATH` (`migrations`)
- `MINIO_BUCKET` (`cloudstore`)
- `MINIO_PRESIGN_TTL_MIN` (`15`)
- `REDIS_POOL_SIZE` (`20`)
- `REDIS_MIN_IDLE_CONNS` (`5`)
- `REDIS_TIMEOUT_MS` (`5000`)

## Локальный запуск

### Вариант 1: полностью в Docker (рекомендуется)

1. Поднять инфраструктуру и backend:

```bash
docker compose up --build
```

2. Проверить health:

```bash
curl http://localhost:8080/health
```

Ожидаемый ответ: `OK`.

MinIO Console: `http://localhost:9001`

### Вариант 2: сервис локально, зависимости в Docker

1. Запустить только инфраструктуру:

```bash
docker compose up -d postgres redis minio
```

2. Применить миграции:

```bash
make migrate-up
```

3. Запустить backend локально:

```bash
make run
```

4. Проверить сервис:

```bash
curl http://localhost:8080/health
```

## Полезные команды

```bash
make run         # запуск backend
make build       # сборка проекта
make test        # тесты
make migrate-up  # применить миграции
make migrate-down # откатить 1 миграцию
```

## Текущий статус

Фаза 1 backend почти завершена, следующий этап - развитие API для работы с файлами, папками и шарингом.
