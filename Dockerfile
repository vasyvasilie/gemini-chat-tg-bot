# Stage 1: Builder
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN GOOS=linux go build -ldflags '-s -w' -a -o gemini-chat cmd/main.go

# Stage 2: Runner
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/gemini-chat .

# Expose any necessary ports if your application listens on one
# For a Telegram bot, typically no ports need to be exposed for incoming connections
# EXPOSE 8080

CMD ["./gemini-chat"]
