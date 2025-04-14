# Этап сборки
FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# ВАЖНО: Статическая сборка для Alpine
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o auth-service

# Финальный минимальный образ
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/auth-service .

EXPOSE 8080

ENTRYPOINT ["./auth-service"]
