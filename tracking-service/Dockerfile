FROM golang:1.18 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o tracking-service

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/tracking-service .
EXPOSE 8080
ENTRYPOINT ["./tracking-service"]
