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
    -o /app/bin/connector-notifications \
    ./cmd/connector-notifications



FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/bin/connector-notifications /app/connector-notifications
COPY --from=builder /app/connectors/notifications/VERSION /app/VERSION

CMD ["/app/connector-notifications"]
