FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY backend ./backend
RUN go build -o /bin/cloudstore-server ./backend/cmd/server

FROM alpine:3.22

WORKDIR /app
COPY --from=builder /bin/cloudstore-server /app/server

EXPOSE 8080

CMD ["/app/server"]
