FROM golang:1.21.4-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o user-service .

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/user-service .

# Expose the port the service listens on
EXPOSE 8001

CMD ["./user-service"]