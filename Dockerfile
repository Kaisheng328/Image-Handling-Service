
FROM golang:1.23-alpine

COPY go.mod go.sum ./
COPY halogen-device-438608-v9-firebase-adminsdk-kwtb8-780d822bbb.json .
RUN go mod download


COPY . .

RUN go build -o main .


EXPOSE 5000

CMD ["./main"]
