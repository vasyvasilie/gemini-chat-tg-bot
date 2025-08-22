package config

import (
	"os"
	"strconv"
	"strings"
)

const (
	defaultModel string = "models/gemini-2.0-flash-lite"
)

type Config struct {
	BotToken     string
	GeminiApiKey string
	AllowedUsers map[int64]struct{}
	StoragePath  string
	DefaultModel string
	Debug        bool
}

func Load() (*Config, error) {
	botToken := os.Getenv("BOT_API_TOKEN")
	if botToken == "" {
		return nil, ErrMissingEnv("BOT_API_TOKEN")
	}

	geminiApiKey := os.Getenv("GEMINI_API_KEY")
	if geminiApiKey == "" {
		return nil, ErrMissingEnv("GEMINI_API_KEY")
	}

	allowedUsersStr := os.Getenv("ALLOWED_USERS")
	if allowedUsersStr == "" {
		return nil, ErrMissingEnv("ALLOWED_USERS")
	}

	storagePath := os.Getenv("STORAGE_PATH")
	if storagePath == "" {
		return nil, ErrMissingEnv("STORAGE_PATH")
	}

	var debug bool
	if debugEnv := os.Getenv("TG_BOT_DEBUG"); debugEnv == "true" {
		debug = true
	}

	allowedUsers := make(map[int64]struct{})
	for _, userIDStr := range strings.Split(allowedUsersStr, ",") {
		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			return nil, ErrInvalidUserID(userIDStr)
		}
		allowedUsers[userID] = struct{}{}
	}

	return &Config{
		BotToken:     botToken,
		GeminiApiKey: geminiApiKey,
		AllowedUsers: allowedUsers,
		StoragePath:  storagePath,
		DefaultModel: defaultModel,
		Debug:        debug,
	}, nil
}

type ErrMissingEnv string

func (e ErrMissingEnv) Error() string {
	return string(e) + " environment variable not set"
}

type ErrInvalidUserID string

func (e ErrInvalidUserID) Error() string {
	return "invalid user ID in ALLOWED_USERS: " + string(e)
}
