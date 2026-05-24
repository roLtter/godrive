# godrive

> Self-hosted cloud file storage. Upload, organize, and share files — your data, your server.

[Русская версия](README.md)

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?style=flat&logo=postgresql&logoColor=white)
![MinIO](https://img.shields.io/badge/MinIO-S3--compatible-C72E49?style=flat&logo=minio&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=flat&logo=redis&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=flat&logo=docker&logoColor=white)

---

## Status

| Phase | Description | Status |
|-------|-------------|--------|
| 1 — Foundation | Docker setup, migrations, health check | ✅ Done |
| 2 — Auth | JWT, refresh tokens, rate limiting | ✅ Done |
| 3 — File Core | Upload, download, folders, quota | ✅ Done |
| 4 — Sharing | Public links, TTL, passwords | ⏳ Planned |
| 5 — Frontend | React TypeScript UI | ⏳ Planned |
| 6 — Deploy | Production config, CI/CD | ⏳ Planned |

---

## Project Structure

```
godrive/
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go
│   ├── internal/
│   │   ├── auth/           # register, login, JWT, bcrypt, refresh tokens
│   │   ├── files/          # upload, download, list, patch, soft delete, trash
│   │   ├── folders/        # CRUD, breadcrumbs, resolve path
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
├── migrations/
│   ├── 000001_init.up.sql
│   ├── 000002_create_users.up.sql
│   ├── 000003_create_folders.up.sql
│   ├── 000004_create_files.up.sql
│   ├── 000005_create_shares.up.sql
│   ├── 000006_add_user_storage_quota.up.sql
│   └── 000007_files_soft_delete.up.sql
├── docker-compose.yml
├── Makefile
├── go.mod
└── go.sum
```

Go module name: `cloudstore`.

---

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.25+ (for local development without Docker)

### Run with Docker

```bash
git clone git@github.com:roLtter/godrive.git
cd godrive

docker compose up -d --build
```

Verify:

```bash
curl http://localhost:8080/health
# OK
```

### Services

| Service | URL |
|---------|-----|
| API | http://localhost:8080 |
| MinIO Console | http://localhost:9001 |
| MinIO API | http://localhost:9000 |
| PostgreSQL | localhost:5432 |
| Redis | localhost:6379 |

### Default credentials

| Service | User | Password |
|---------|------|----------|
| PostgreSQL | cloudstore | cloudstore |
| Redis | — | cloudstore |
| MinIO | cloudstore | cloudstore123 |

### Local run (without Docker backend)

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

---

## API Reference

Auth endpoints are public. All `/api/*` endpoints require:

`Authorization: Bearer <access_token>`

### Auth

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/register` | Create account |
| POST | `/login` | Login, receive tokens |
| POST | `/refresh` | Refresh access token |
| POST | `/logout` | Invalidate refresh token |

### Folders

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/folders` | Create folder |
| GET | `/api/folders?parent_id=<id>` | List child folders (root if `parent_id` omitted) |
| GET | `/api/folders/resolve?path=/a/b/c` | Resolve folder by path + breadcrumbs |
| GET | `/api/folders/:id/breadcrumbs` | Breadcrumbs for folder |
| PATCH | `/api/folders/:id` | Rename folder |
| DELETE | `/api/folders/:id` | Delete folder |

### Files

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/upload` | Upload file (multipart) |
| GET | `/api/files` | List files (paginated) |
| GET | `/api/files/trash` | List soft-deleted files |
| GET | `/api/download?file_id=<id>` | Download via presigned URL (302 redirect) |
| PATCH | `/api/files/:id` | Rename or move file |
| DELETE | `/api/files/:id` | Soft delete |

### System

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Service health status (plain text `OK`) |
| GET | `/api/me` | Current user from JWT |

### Example: Upload a file

```bash
# 1. Login
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'

# 2. Create folder (folder_id is required for upload)
curl -X POST http://localhost:8080/api/folders \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Documents"}'

# 3. Upload
curl -X POST http://localhost:8080/api/upload \
  -H "Authorization: Bearer <access_token>" \
  -F "file=@document.pdf" \
  -F "folder_id=1"
```

Response (201):

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

### Example: Download a file

```bash
curl -L "http://localhost:8080/api/download?file_id=42" \
  -H "Authorization: Bearer <access_token>"
```

---

## Database Schema

```
users       — id (UUID), email, password_hash, created_at,
              storage_quota_bytes, storage_used_bytes

folders     — id (BIGSERIAL), user_id, parent_id, name

files       — id (BIGSERIAL), user_id, folder_id, name, size, mime, s3_key,
              created_at, deleted_at

shares      — id (BIGSERIAL), file_id, token, expires_at, password_hash
              (table exists; API not implemented yet)
```

---

## Development

### Makefile commands

```bash
make run          # start backend server
make build        # build all packages
make test         # run all tests
make migrate-up   # apply migrations
make migrate-down # rollback last migration
```

### Running tests

```bash
# from repo root
go test ./...

go test ./backend/internal/auth/...
go test ./backend/internal/files/...
```
