FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY backend ./backend
COPY migrations ./migrations
RUN go build -o /bin/cloudstore-server ./backend/cmd/server \
	&& go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.3

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /bin/cloudstore-server /app/server
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate
COPY --from=builder /app/migrations /app/migrations
COPY backend/docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh /usr/local/bin/migrate /app/server

EXPOSE 8080

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD []
