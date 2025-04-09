FROM golang:1.24 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o order-service

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/order-service .
EXPOSE 8080
ENTRYPOINT ["./order-service"]
