FROM golang:1.24-alpine

WORKDIR /app

COPY go.mod ./go.mod
COPY go.sum ./go.sum
RUN go mod download

COPY . .

RUN go build -o /api_gateway main.go

EXPOSE 8080

CMD ["/api_gateway"]
