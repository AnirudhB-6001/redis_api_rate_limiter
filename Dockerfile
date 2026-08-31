FROM golang:1.27-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./

RUN go build -o rate-limiter .

CMD ["./rate-limiter"]