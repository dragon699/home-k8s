FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY common-go/go.mod common-go/go.sum /app/common-go/
COPY common-go /app/common-go

COPY connectors/notifications/go.mod connectors/notifications/go.sum /app/connectors/notifications/

WORKDIR /app/connectors/notifications

RUN go mod download

COPY connectors/notifications /app/connectors/notifications

RUN CGO_ENABLED=0 \
    GOOS=linux \
    go build \
    -o /app/bin/notifications-controller \
    ./cmd/notifications



FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/bin/notifications-controller /app/notifications-controller
COPY --from=builder /app/connectors/notifications/VERSION /app/VERSION

CMD ["/app/notifications-controller"]
