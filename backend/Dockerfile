FROM golang:1.25-alpine AS builder

# If build fails with sum.golang.org timeout, build with: --build-arg GOSUMDB=off
ARG GOSUMDB=sum.golang.org
ENV GOSUMDB=${GOSUMDB}

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY backend ./backend
COPY migrations ./migrations
# go build uses modules from go.sum (already downloaded); go install @version often re-hits sum.golang.org
RUN go build -o /bin/cloudstore-server ./backend/cmd/server \
	&& go build -trimpath -tags postgres -o /go/bin/migrate github.com/golang-migrate/migrate/v4/cmd/migrate

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
