FROM golang:1.22.1

LABEL maintainer="Tom Humphris <tom@muska.co.uk>"

WORKDIR /app

COPY . .

RUN go mod download

RUN go build -o main ./

CMD ["./main"]
