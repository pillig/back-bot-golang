# syntax=docker/dockerfile:1

FROM golang:1.22.3

WORKDIR /app

COPY . .

RUN go mod download

# Build
RUN go build -o /back-bot

CMD ["/back-bot -f super-secret-token.txt"]