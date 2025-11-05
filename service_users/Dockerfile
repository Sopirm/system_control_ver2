FROM golang:1.21-alpine

WORKDIR /app

COPY go.mod ./go.mod
COPY go.sum ./go.sum
RUN go mod download

COPY . .

RUN go build -o /service_users main.go

EXPOSE 8081

CMD ["/service_users"]
