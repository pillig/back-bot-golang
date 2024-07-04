# syntax=docker/dockerfile:1

FROM golang:1.22.3-alpine AS builder

WORKDIR /app

COPY . .

# Build
RUN go build -o back-bot

FROM alpine

WORKDIR /app

COPY --from=builder /app/back_repo ./back_repo
COPY --from=builder /app/back-bot .

CMD ["/bin/sh", "-c", "./back-bot -t $TOKEN"]
