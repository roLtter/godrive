# godrive

> Self-hosted облачное хранилище файлов.

[English version](README.en.md)

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go)
![React](https://img.shields.io/badge/React-19-61DAFB?style=flat&logo=react&logoColor=black)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?style=flat&logo=postgresql&logoColor=white)
![MinIO](https://img.shields.io/badge/MinIO-S3--compatible-C72E49?style=flat&logo=minio&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=flat&logo=redis&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=flat&logo=docker&logoColor=white)

---

## Статус

| Фаза | Описание | Статус |
|------|----------|--------|
| 1 — Foundation | Docker, миграции, health check | ✅ Готово |
| 2 — Auth | JWT, refresh tokens, rate limiting | ✅ Готово |
| 3 — File Core | Upload, download, папки, квота, trash | ✅ Готово |
| 4 — Sharing | Публичные ссылки, TTL, пароли, счётчик скачиваний | ✅ Готово |
| 5 — Frontend | React + TypeScript UI | ⏳ В разработке |
| 6 — Deploy | Production config, CI/CD, frontend в Compose | ⏳ Запланировано |

**Frontend сейчас:** логин/регистрация, страница Drive с загрузкой (выбор/создание папки) и списком файлов (grid/list). Shared links и Trash — заглушки (API уже есть).

---

## Структура проекта

```
godrive/
├── backend/
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── auth/           # register, login, JWT, bcrypt, refresh tokens
│   │   ├── files/          # upload, download, list, patch, soft delete, trash
│   │   ├── folders/        # CRUD, breadcrumbs, resolve path
│   │   ├── shares/         # create, list, revoke, public resolve
│   │   ├── storage/minio/  # MinIO wrapper, presigned URLs
│   │   ├── cache/redis/    # Redis client, pool
│   │   ├── db/postgres/    # PostgreSQL client
│   │   ├── middleware/     # JWT auth, rate limiter, request logger
│   │   ├── config/         # env-based config (viper)
│   │   ├── logger/         # zap logger
│   │   ├── worker/         # cleanup soft-deleted files from MinIO
│   │   └── dbmigrate/      # pin migrate drivers
│   ├── docker-entrypoint.sh
│   └── Dockerfile
├── frontend/               # React 19 + Vite + TypeScript + Tailwind
│   ├── src/
│   │   ├── api/            # auth, files, folders, me
│   │   ├── pages/          # Drive, Login, Register, Shared, Trash
│   │   ├── components/
│   │   └── contexts/
│   └── Dockerfile          # dev-образ (пока не в docker-compose)
├── migrations/
│   ├── 000001_init.up.sql
│   ├── 000002_create_users.up.sql
│   ├── 000003_create_folders.up.sql
│   ├── 000004_create_files.up.sql
│   ├── 000005_create_shares.up.sql
│   ├── 000006_add_user_storage_quota.up.sql
│   ├── 000007_files_soft_delete.up.sql
│   └── 000008_shares_access_tracking.up.sql
├── docker-compose.yml      # postgres, redis, minio, backend
├── Makefile
├── go.mod
└── go.sum
```

Имя Go-модуля: `cloudstore`.

---

## Быстрый старт

### Требования

- Docker и Docker Compose
- Go 1.25+ (для локальной разработки backend без Docker)
- Node.js 22+ (для frontend)

### Запуск инфраструктуры и API в Docker

```bash
git clone git@github.com:roLtter/godrive.git
cd godrive

docker compose up -d --build
```

Миграции применяются автоматически при старте backend.

Проверка:

```bash
curl http://localhost:8080/health
# OK
```

### Сервисы

| Сервис | URL |
|--------|-----|
| API | http://localhost:8080 |
| MinIO Console | http://localhost:9001 |
| MinIO API | http://localhost:9000 |
| PostgreSQL | localhost:5432 |
| Redis | localhost:6379 |

Frontend в `docker-compose` пока не включён — запускается отдельно (см. ниже).

### Учётные данные по умолчанию

| Сервис | Пользователь | Пароль |
|--------|--------------|--------|
| PostgreSQL | cloudstore | cloudstore |
| Redis | — | cloudstore |
| MinIO | cloudstore | cloudstore123 |

### Локальный запуск backend (без Docker-образа API)

```bash
docker compose up -d postgres redis minio

export JWT_SECRET=dev-secret-change-me-in-production
export DB_URL=postgres://cloudstore:cloudstore@127.0.0.1:5432/cloudstore?sslmode=disable
export REDIS_URL=redis://:cloudstore@127.0.0.1:6379/0
export MINIO_URL=http://127.0.0.1:9000
export MINIO_ROOT_USER=cloudstore
export MINIO_ROOT_PASSWORD=cloudstore123

make migrate-up
make run
```

### Локальный запуск frontend

```bash
cd frontend
npm install
npm run dev
```

UI: http://localhost:5173

В dev Vite проксирует `/api`, `/login`, `/register`, `/refresh`, `/logout` на `http://localhost:8080`.  
Опционально: `VITE_API_BASE_URL` для прямого указания базового URL API.

---

## API

Публичные: auth, health и публичные шары. Остальные `/api/*` требуют:

`Authorization: Bearer <access_token>`

### Auth

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/register` | Регистрация (пароль ≥ 8 символов) |
| POST | `/login` | Вход, получение токенов |
| POST | `/refresh` | Обновление access token |
| POST | `/logout` | Выход, инвалидация refresh token |

### Папки

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/api/folders` | Создать папку |
| GET | `/api/folders?parent_id=<id>` | Список дочерних папок (корень — без `parent_id`) |
| GET | `/api/folders/resolve?path=/a/b/c` | Папка по пути + breadcrumbs |
| GET | `/api/folders/:id` | Папка по id |
| GET | `/api/folders/:id/breadcrumbs` | Breadcrumbs для папки |
| PATCH | `/api/folders/:id` | Переименовать папку |
| DELETE | `/api/folders/:id` | Удалить папку |

### Файлы

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/api/files/upload` | Загрузка файла (multipart: `file`, `folder_id`) |
| GET | `/api/files` | Список файлов (пагинация, `folder_id`, `sort_by`, `sort_order`) |
| GET | `/api/files/trash` | Список удалённых файлов |
| GET | `/api/files/:id/download` | Скачивание через presigned URL (302) |
| PATCH | `/api/files/:id` | Переименовать или переместить файл |
| DELETE | `/api/files/:id` | Мягкое удаление |

По умолчанию: лимит загрузки **20 MB**, MIME: `image/jpeg`, `image/png`, `image/webp`, `application/pdf`, `text/plain` (настраивается через env).

### Шаринг

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/api/shares` | Создать публичную ссылку (`file_id`, опционально `ttl_seconds` / `expires_in`, `password`) |
| GET | `/api/shares` | Активные (не истёкшие) ссылки текущего пользователя |
| DELETE | `/api/shares/:id` | Отозвать ссылку |
| GET | `/s/:token` | Публичный доступ (302 на MinIO); при пароле — `?password=` |

TTL по умолчанию: 30 дней, максимум: 365 дней. Пароль шары (если задан) — не короче 4 символов.

### Система

| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/health` | Статус сервиса (текст `OK`) |
| GET | `/api/me` | Текущий пользователь из JWT |

### Пример: загрузка файла

```bash
# 1. Вход
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'

# 2. Создать папку (folder_id обязателен для upload)
curl -X POST http://localhost:8080/api/folders \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Documents"}'

# 3. Загрузка
curl -X POST http://localhost:8080/api/files/upload \
  -H "Authorization: Bearer <access_token>" \
  -F "file=@document.pdf" \
  -F "folder_id=1"
```

Ответ (201):

```json
{
  "id": 42,
  "folder_id": 1,
  "name": "document.pdf",
  "size": 204800,
  "mime": "application/pdf",
  "s3_key": "users/<user_uuid>/.../document.pdf",
  "created_at": "2026-05-24T12:00:00Z"
}
```

### Пример: скачивание файла

```bash
curl -L "http://localhost:8080/api/files/42/download" \
  -H "Authorization: Bearer <access_token>"
```

### Пример: публичная ссылка

```bash
curl -X POST http://localhost:8080/api/shares \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"file_id":42,"ttl_seconds":86400,"password":"secret"}'

# скачать по токену (с паролем)
curl -L "http://localhost:8080/s/<token>?password=secret"
```

---

## Схема базы данных

```
users       — id (UUID), email, password_hash, created_at,
              storage_quota_bytes (default 5 GiB), storage_used_bytes

folders     — id (BIGSERIAL), user_id, parent_id, name

files       — id (BIGSERIAL), user_id, folder_id, name, size, mime, s3_key,
              created_at, deleted_at

shares      — id (BIGSERIAL), file_id, token, expires_at, password_hash,
              download_count, last_accessed_at
```

Фоновый worker периодически удаляет объекты MinIO для файлов с `deleted_at` старше `TRASH_MIN_AGE_MINUTES` (по умолчанию 24 часа / 1440 минут).

---

## Разработка

### Команды Makefile

```bash
make run          # запуск backend
make build        # сборка всех пакетов
make test         # все тесты
make migrate-up   # применить миграции
make migrate-down # откатить последнюю миграцию
```

### Запуск тестов

```bash
# из корня репозитория
go test ./...

go test ./backend/internal/auth/...
go test ./backend/internal/files/...
go test ./backend/internal/shares/...
```

### Frontend

```bash
cd frontend
npm run dev       # Vite dev-server
npm run build     # production build
npm run lint      # ESLint
```
