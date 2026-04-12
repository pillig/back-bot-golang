FROM golang:1.22.3-alpine AS builder

WORKDIR /app

COPY . .

RUN go get github.com/bwmarrin/discordgo@master && \
    go mod tidy && \
    go build -o back-bot

FROM alpine

WORKDIR /app

COPY --from=builder /app/back_repo ./back_repo
COPY --from=builder /app/back-bot .

ENTRYPOINT ["./back-bot"]
