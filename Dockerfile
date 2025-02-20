
FROM golang:1.23-alpine

COPY go.mod go.sum ./

RUN go mod download


COPY . .

RUN go build -o main .


EXPOSE 5000

CMD ["./main"]
