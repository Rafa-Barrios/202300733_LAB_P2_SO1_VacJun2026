FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY src/ ./src/

RUN go build -o go-consumer ./src/

FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/go-consumer .

ENV RABBITMQ_URL=amqp://admin:admin123@rabbitmq.sopes1.svc.cluster.local:5672/
ENV VALKEY_ADDR=valkey-service.sopes1.svc.cluster.local:6379

CMD ["./go-consumer"]