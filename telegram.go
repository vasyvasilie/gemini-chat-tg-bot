package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"sync"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
)

const (
	defaultModel string = "models/gemini-1.5-flash" 
)

// UserSettings holds individual settings for each user.
type UserSettings struct {
	ModelName string
}

// UserSession holds all session data for a single user.
type UserSession struct {
	Chat     *Chat
	Settings UserSettings
}

// sessions is the main in-memory storage for user sessions.
var sessions = make(map[int64]*UserSession)
var sessionsMu sync.RWMutex

// allowedUsers stores the set of users who are allowed to use the bot.
var allowedUsers = make(map[int64]bool)

// StartBot initializes and starts the Telegram bot.
func StartBot() {
	botToken := os.Getenv("BOT_API_TOKEN")
	if botToken == "" {
		log.Fatal("BOT_API_TOKEN environment variable not set")
	}

	geminiApiKey := os.Getenv("GEMINI_API_KEY")
	if geminiApiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable not set")
	}

	allowedUsersStr := os.Getenv("ALLOWED_USERS")
	if allowedUsersStr == "" {
		log.Fatal("ALLOWED_USERS environment variable not set")
	}
	for _, userIDStr := range strings.Split(allowedUsersStr, ",") {
		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			log.Printf("Invalid user ID in ALLOWED_USERS: %s", userIDStr)
			continue
		}
		allowedUsers[userID] = true
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	bot, err := telego.NewBot(botToken, telego.WithDefaultLogger(false, true))
	//bot, err := telego.NewBot(botToken, telego.WithDefaultLogger(true, true))
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Set bot commands for Telegram UI
	_ = bot.SetMyCommands(ctx, &telego.SetMyCommandsParams{
		Commands: []telego.BotCommand{
			{Command: "new", Description: "Start a new chat session"},
			{Command: "listmodels", Description: "Show available models"},
			{Command: "setmodel", Description: "Set a model (e.g. /setmodel model-name)"},
			{Command: "currentmodel", Description: "Show the currently selected model"},
		},
	})

	updates, err := bot.UpdatesViaLongPolling(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to get updates: %v", err)
	}

	bh, err := th.NewBotHandler(bot, updates)
	if err != nil {
		log.Fatalf("Failed to create bot handler: %v", err)
	}
	defer bh.Stop()

	// Middleware to check if the user is allowed
	bh.Use(func(ctx *th.Context, update telego.Update) error {
		var userID int64
		if update.Message != nil {
			userID = update.Message.From.ID
		} else {
			return ctx.Next(update)
		}

		if !allowedUsers[userID] {
			log.Printf("Unauthorized access from user ID: %d", userID)
			return nil // Stop processing if user is not allowed
		}
		return ctx.Next(update)
	})

        bh.Handle(func(ctx *th.Context, update telego.Update) error {
                userID := update.Message.From.ID
		modelName := defaultModel

                sessionsMu.RLock()
                session, ok := sessions[userID]
                sessionsMu.RUnlock()

                if ok {
                        modelName = session.Settings.ModelName
                }

                newChat, err := NewChat(ctx, modelName)
                if err != nil {
                        log.Printf("Failed to create new chat for user %d: %v", userID, err)
                        _, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Failed to start a new session. Please try again."))
                        return err
                }

                sessionsMu.Lock()
                if session == nil { // This case should ideally not happen if 'ok' was true, but for safety
                    session = &UserSession{}
                    sessions[userID] = session
                }
                session.Chat = newChat
                sessionsMu.Unlock()

                log.Printf("Started new session for user %d with model %s", userID, session.Settings.ModelName)
                _, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "✅ New chat session started. Your history has been cleared."))
                return nil
        }, th.CommandEqual("new"))

	// Handler for /start command
	bh.Handle(func(ctx *th.Context, update telego.Update) error {
		userID := update.Message.From.ID
		modelName := defaultModel

		sessionsMu.RLock()
		session, ok := sessions[userID]
		sessionsMu.RUnlock()

		if !ok {
			log.Printf("First message from user %d via /start, creating new session.", userID)
			settings := UserSettings{ModelName: modelName}
			newChat, err := NewChat(ctx, settings.ModelName)
			if err != nil {
				log.Printf("Failed to create new chat for new user %d: %v", userID, err)
				_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Failed to initialize session. Please try again."))
				return err
			}
			session = &UserSession{Chat: newChat, Settings: settings}
			sessionsMu.Lock()
			sessions[userID] = session
			sessionsMu.Unlock()
		} else {
			modelName = session.Settings.ModelName
		}

		response := "👋 Welcome! Your current model is: `" + modelName + "`"
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), response).WithParseMode(telego.ModeMarkdown))
		return nil
	}, th.CommandEqual("start"))

	// Handler for /listmodels command
	bh.Handle(func(ctx *th.Context, update telego.Update) error {
		userID := update.Message.From.ID
		models, err := ListModels(ctx)
		if err != nil {
			log.Printf("Failed to list models for user %d: %v", userID, err)
			_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Sorry, I couldn't retrieve the list of models."))
			return err
		}
		
		response := "✨ Available models:\n`" + strings.Join(models, "`\n`") + "`"
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), response).WithParseMode(telego.ModeMarkdown))
		return nil
	}, th.CommandEqual("listmodels"))

	// Handler for /setmodel command
	bh.Handle(func(ctx *th.Context, update telego.Update) error {
		userID := update.Message.From.ID
		parts := strings.Fields(update.Message.Text)
		if len(parts) < 2 {
			_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "⚠️ Please specify a model name.\nUsage: `/setmodel model-name`").WithParseMode(telego.ModeMarkdown))
			return nil
		}
		modelName := parts[1]

		sessionsMu.RLock()
		session, ok := sessions[userID]
		sessionsMu.RUnlock()

		if !ok {
			log.Printf("First message from user %d via /setmodel, creating new session.", userID)
			settings := UserSettings{ModelName: modelName}
			newChat, err := NewChat(ctx, settings.ModelName)
			if err != nil {
				log.Printf("Failed to create new chat for new user %d: %v", userID, err)
				_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Failed to initialize session with the new model."))
				return err
			}
			session = &UserSession{Chat: newChat, Settings: settings}
			sessionsMu.Lock()
			sessions[userID] = session
			sessionsMu.Unlock()
		} else {
			sessionsMu.Lock()
			session.Settings.ModelName = modelName
			sessionsMu.Unlock()

			newChat, err := NewChat(ctx, modelName)
			if err != nil {
				log.Printf("Failed to create new chat for user %d with new model: %v", userID, err)
				_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Failed to switch to the new model. Please try again."))
				return err
			}
			sessionsMu.Lock()
			session.Chat = newChat
			sessionsMu.Unlock()
		}

		log.Printf("Set model for user %d to %s", userID, modelName)
		response := "✅ Model successfully changed to `" + modelName + "`. A new chat session has been started."
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), response).WithParseMode(telego.ModeMarkdown))
		return nil
	}, th.CommandEqual("setmodel"))

	// Handler for /currentmodel command
	bh.Handle(func(ctx *th.Context, update telego.Update) error {
		userID := update.Message.From.ID
		modelName := defaultModel

		sessionsMu.RLock()
		session, ok := sessions[userID]
		sessionsMu.RUnlock()

		if ok {
			modelName = session.Settings.ModelName
		}

		response := "✨ Your current model is: `" + modelName + "`"
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), response).WithParseMode(telego.ModeMarkdown))
		return nil
	}, th.CommandEqual("currentmodel"))

	// Handler for all other messages
	bh.Handle(func(ctx *th.Context, update telego.Update) error {
		userID := update.Message.From.ID
		text := update.Message.Text

		session, ok := sessions[userID]
		if !ok {
			log.Printf("First message from user %d, creating new session.", userID)
			settings := UserSettings{ModelName: defaultModel}
			newChat, err := NewChat(ctx, settings.ModelName)
			if err != nil {
				log.Printf("Failed to create new chat for new user %d: %v", userID, err)
				return err
			}
			session = &UserSession{Chat: newChat, Settings: settings}
			sessions[userID] = session
			_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "👋 New session created with default model: `" + settings.ModelName + "`").WithParseMode(telego.ModeMarkdown))
		}

		_ = ctx.Bot().SendChatAction(ctx, &telego.SendChatActionParams{
			ChatID: tu.ID(userID),
			Action: telego.ChatActionTyping,
		})

		response, err := session.Chat.GenerateContent(ctx, text)
		if err != nil {
			log.Printf("Failed to get response from Gemini for user %d: %v", userID, err)
			errorMessage := "Sorry, I couldn't get a response. Please try again."
			if strings.Contains(err.Error(), "429:") {
				errorMessage = "❌ Gemini API rate limit exceeded. Please try again later."
			}
			_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), errorMessage))
			return err
		}

		_, err = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), response).WithParseMode(telego.ModeMarkdown))
		if err != nil {
			log.Printf("Failed to send message to user %d: %v", userID, err)
		}
		return err
	}, th.AnyMessage())

	log.Println("Bot starting...")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		bh.Start()
	}()

	<-ctx.Done()
	log.Println("Stopping bot...")
	bh.Stop()
	wg.Wait()
	log.Println("Bot stopped.")
}
