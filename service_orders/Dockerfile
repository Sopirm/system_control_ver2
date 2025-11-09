FROM golang:1.21-alpine

WORKDIR /app

COPY go.mod ./go.mod
COPY go.sum ./go.sum
RUN go mod download

COPY . .

RUN go mod tidy
RUN go build -o /service_orders main.go

EXPOSE 8082

CMD ["/service_orders"]
