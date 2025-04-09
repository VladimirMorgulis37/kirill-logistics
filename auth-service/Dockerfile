# Этап сборки
FROM golang:1.24 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o auth-service

# Финальный образ
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/auth-service .
EXPOSE 8080
ENTRYPOINT ["./auth-service"]
