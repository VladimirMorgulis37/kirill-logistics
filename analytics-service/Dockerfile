# Этап сборки
FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Статическая сборка для Alpine Linux
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o analytics-service

# Финальный образ
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/analytics-service .

EXPOSE 8080

ENTRYPOINT ["./analytics-service"]
