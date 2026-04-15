GO := /usr/local/go/bin/go

MIGRATE_PKG := github.com/golang-migrate/migrate/v4/cmd/migrate
MIGRATE_VER := v4.18.3

MIGRATIONS_PATH ?= migrations
DATABASE_URL ?= postgres://cloudstore:cloudstore@127.0.0.1:5432/cloudstore?sslmode=disable

.PHONY: run build test migrate-up migrate-down

run:
	$(GO) run ./backend/cmd/server

build:
	$(GO) build ./...

test:
	$(GO) test ./...

migrate-up:
	$(GO) run -tags postgres $(MIGRATE_PKG)@$(MIGRATE_VER) -database "$(DATABASE_URL)" -path "$(MIGRATIONS_PATH)" up

migrate-down:
	$(GO) run -tags postgres $(MIGRATE_PKG)@$(MIGRATE_VER) -database "$(DATABASE_URL)" -path "$(MIGRATIONS_PATH)" down 1
