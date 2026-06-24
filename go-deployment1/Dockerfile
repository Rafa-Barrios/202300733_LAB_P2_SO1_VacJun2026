FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY proto/ ./proto/
COPY src/ ./src/

RUN go build -o go-deployment1 ./src/

FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/go-deployment1 .

EXPOSE 8080

ENV PORT=8080
ENV GIN_MODE=release

CMD ["./go-deployment1"]