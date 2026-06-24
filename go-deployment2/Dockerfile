FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY proto/ ./proto/
COPY src/ ./src/

RUN go build -o go-deployment2 ./src/

FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/go-deployment2 .

EXPOSE 50051

ENV GRPC_PORT=50051

CMD ["./go-deployment2"]