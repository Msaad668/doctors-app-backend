FROM golang:alpine

WORKDIR /app

COPY . .

RUN go build -o apigateway .

CMD ["/app/apigateway"]