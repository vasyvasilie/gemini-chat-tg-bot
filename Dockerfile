# Stage 1: Builder
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -gcflags="all=-N -l" -a -o gemini-chat .

# Stage 2: Runner
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/gemini-chat .

# Expose any necessary ports if your application listens on one
# For a Telegram bot, typically no ports need to be exposed for incoming connections
# EXPOSE 8080

CMD ["./gemini-chat"]
