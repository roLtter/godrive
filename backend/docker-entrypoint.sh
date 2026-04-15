#!/bin/sh
set -eu

MIGRATIONS_PATH="${MIGRATIONS_PATH:-/app/migrations}"

if [ -z "${DB_URL:-}" ]; then
	echo "docker-entrypoint: DB_URL is not set" >&2
	exit 1
fi

echo "docker-entrypoint: applying migrations from ${MIGRATIONS_PATH}"
/usr/local/bin/migrate -path "$MIGRATIONS_PATH" -database "$DB_URL" up

echo "docker-entrypoint: starting server"
exec /app/server "$@"
