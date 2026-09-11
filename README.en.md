# godrive

> Self-hosted cloud file storage.

[Русская версия](README.md)

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go)
![React](https://img.shields.io/badge/React-19-61DAFB?style=flat&logo=react&logoColor=black)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?style=flat&logo=postgresql&logoColor=white)
![MinIO](https://img.shields.io/badge/MinIO-S3--compatible-C72E49?style=flat&logo=minio&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=flat&logo=redis&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=flat&logo=docker&logoColor=white)

---

## Status

| Phase | Description | Status |
|-------|-------------|--------|
| 1 — Foundation | Docker, migrations, health check | ✅ Done |
| 2 — Auth | JWT, refresh tokens, rate limiting | ✅ Done |
| 3 — File Core | Upload, download, folders, quota, trash | ✅ Done |
| 4 — Sharing | Public links, TTL, passwords, download counter | ✅ Done |
| 5 — Frontend | React + TypeScript UI | ⏳ In progress |
| 6 — Deploy | Production config, CI/CD, frontend in Compose | ⏳ Planned |

**Frontend today:** login/register, Drive page with folder-aware upload and file list (grid/list). Shared links and Trash pages are placeholders (backend API already exists).

---

## Project structure

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
│   └── Dockerfile          # dev image (not in docker-compose yet)
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

Go module name: `cloudstore`.

---

## Quick start

### Prerequisites

- Docker and Docker Compose
- Go 1.25+ (for local backend development without Docker)
- Node.js 22+ (for frontend)

### Run infrastructure and API with Docker

```bash
git clone git@github.com:roLtter/godrive.git
cd godrive

docker compose up -d --build
```

Migrations run automatically when the backend starts.

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

Frontend is not in `docker-compose` yet — run it separately (see below).

### Default credentials

| Service | User | Password |
|---------|------|----------|
| PostgreSQL | cloudstore | cloudstore |
| Redis | — | cloudstore |
| MinIO | cloudstore | cloudstore123 |

### Local backend (without the API Docker image)

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

### Local frontend

```bash
cd frontend
npm install
npm run dev
```

UI: http://localhost:5173

In dev, Vite proxies `/api`, `/login`, `/register`, `/refresh`, `/logout` to `http://localhost:8080`.  
Optional: `VITE_API_BASE_URL` to point at the API directly.

---

## API

Public: auth, health, and public shares. All other `/api/*` routes require:

`Authorization: Bearer <access_token>`

### Auth

| Method | Path | Description |
|--------|------|-------------|
| POST | `/register` | Create account (password ≥ 8 characters) |
| POST | `/login` | Login, receive tokens |
| POST | `/refresh` | Refresh access token |
| POST | `/logout` | Logout, invalidate refresh token |

### Folders

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/folders` | Create folder |
| GET | `/api/folders?parent_id=<id>` | List child folders (root if `parent_id` omitted) |
| GET | `/api/folders/resolve?path=/a/b/c` | Resolve folder by path + breadcrumbs |
| GET | `/api/folders/:id` | Get folder by id |
| GET | `/api/folders/:id/breadcrumbs` | Breadcrumbs for folder |
| PATCH | `/api/folders/:id` | Rename folder |
| DELETE | `/api/folders/:id` | Delete folder |

### Files

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/files/upload` | Upload file (multipart: `file`, `folder_id`) |
| GET | `/api/files` | List files (pagination, `folder_id`, `sort_by`, `sort_order`) |
| GET | `/api/files/trash` | List soft-deleted files |
| GET | `/api/files/:id/download` | Download via presigned URL (302) |
| PATCH | `/api/files/:id` | Rename or move file |
| DELETE | `/api/files/:id` | Soft delete |

Defaults: upload limit **20 MB**; MIME allowlist: `image/jpeg`, `image/png`, `image/webp`, `application/pdf`, `text/plain` (configurable via env).

### Sharing

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/shares` | Create public link (`file_id`, optional `ttl_seconds` / `expires_in`, `password`) |
| GET | `/api/shares` | Active (non-expired) links for current user |
| DELETE | `/api/shares/:id` | Revoke link |
| GET | `/s/:token` | Public access (302 to MinIO); with password use `?password=` |

Default TTL: 30 days; maximum: 365 days. Share password (if set) must be at least 4 characters.

### System

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check (plain text `OK`) |
| GET | `/api/me` | Current user from JWT |

### Example: upload a file

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
curl -X POST http://localhost:8080/api/files/upload \
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

### Example: download a file

```bash
curl -L "http://localhost:8080/api/files/42/download" \
  -H "Authorization: Bearer <access_token>"
```

### Example: public share link

```bash
curl -X POST http://localhost:8080/api/shares \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"file_id":42,"ttl_seconds":86400,"password":"secret"}'

# download by token (with password)
curl -L "http://localhost:8080/s/<token>?password=secret"
```

---

## Database schema

```
users       — id (UUID), email, password_hash, created_at,
              storage_quota_bytes (default 5 GiB), storage_used_bytes

folders     — id (BIGSERIAL), user_id, parent_id, name

files       — id (BIGSERIAL), user_id, folder_id, name, size, mime, s3_key,
              created_at, deleted_at

shares      — id (BIGSERIAL), file_id, token, expires_at, password_hash,
              download_count, last_accessed_at
```

A background worker periodically removes MinIO objects for files with `deleted_at` older than `TRASH_MIN_AGE_MINUTES` (default 24 hours / 1440 minutes).

---

## Development

### Makefile commands

```bash
make run          # start backend
make build        # build all packages
make test         # run all tests
make migrate-up   # apply migrations
make migrate-down # roll back last migration
```

### Running tests

```bash
# from repo root
go test ./...

go test ./backend/internal/auth/...
go test ./backend/internal/files/...
go test ./backend/internal/shares/...
```

### Frontend

```bash
cd frontend
npm run dev       # Vite dev server
npm run build     # production build
npm run lint      # ESLint
```
