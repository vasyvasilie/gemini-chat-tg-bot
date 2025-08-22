# Project Information for Gemini CLI

This file contains project-specific information to assist the Gemini CLI agent.

## Project Name
`gemini-chat-tg-bot`

## Project Root Directory
`/root/go-projects/gemini-chat-tg-bot`

## Main Language
Go

## Go Version Requirement
`go.mod` requires Go version `1.24.4`.

## Build Commands
- Standard Go build: `go build ./...`
- Docker image build: `docker build -t gemini-chat-tg-bot:latest .`

## Docker Run Command
To run the Docker container in the background with environment variables passed from the current shell:
`docker run --name gemini-chat-tg-bot -d -v /tmp/bolt.db:/tmp/bolt.db -e STORAGE_PATH -e BOT_API_TOKEN -e GEMINI_API_KEY -e ALLOWED_USERS gemini-chat-tg-bot:latest`

## Telego Logging Level
The `telego` library is configured to log at `INFO` level using `telego.WithDefaultLogger(false, true)`.
