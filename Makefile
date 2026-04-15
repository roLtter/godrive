GO := /usr/local/go/bin/go

.PHONY: run build test

run:
	$(GO) run ./backend/cmd/server

build:
	$(GO) build ./...

test:
	$(GO) test ./...
