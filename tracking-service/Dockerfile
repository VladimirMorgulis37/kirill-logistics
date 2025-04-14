# Этап сборки
FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# ВАЖНО: статическая сборка для Alpine Linux
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o tracking-service

# Финальный образ
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/tracking-service .

EXPOSE 8080

ENTRYPOINT ["./tracking-service"]
