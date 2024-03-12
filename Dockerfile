FROM golang:latest

LABEL maintainer="Tom Humphris <tom@muska.co.uk>"

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o main .

EXPOSE 8080

CMD ["./main"]
