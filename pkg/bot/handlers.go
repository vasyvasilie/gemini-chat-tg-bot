package bot

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/vasyvasilie/gemini-chat-tg-bot/pkg/gemini"
	"github.com/vasyvasilie/gemini-chat-tg-bot/pkg/storage"
)

func (b *botImpl) setupHandlers(ctx context.Context) {
	b.tgBotHandler.Handle(b.handlerNew, th.CommandEqual("new"))
	b.tgBotHandler.Handle(b.handlerStart, th.CommandEqual("start"))
	b.tgBotHandler.Handle(b.handlerListModels, th.CommandEqual("listmodels"))
	b.tgBotHandler.Handle(b.handlerSetModel, th.CommandEqual("setmodel"))
	b.tgBotHandler.Handle(b.handlerCurrentModel, th.CommandEqual("currentmodel"))
	b.tgBotHandler.Handle(b.handlerDBSize, th.CommandEqual("dbsize"))
	b.tgBotHandler.Handle(b.handlerAnyMessage, th.AnyMessage())
}

// Handler for /new command
func (b *botImpl) handlerNew(ctx *th.Context, update telego.Update) error {
	userID := update.Message.From.ID
	userSettings, err := b.getUserSettings(userID)
	if err != nil {
		log.Printf("Failed to get settings for user %d: %v", userID, err)
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Failed to get user settings."))
		return err
	}

	userSettings.History.Messages = nil
	if err = b.storage.SaveUserSettings(userID, userSettings); err != nil {
		log.Printf("Failed to save settings for user %d: %v", userID, err)
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Failed to save settings."))
		return err
	}

	log.Printf("Started new session for user %d with model %s", userID, userSettings.ModelName)
	_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "✅ New chat session started. Your history has been cleared."))
	return nil
}

// Handler for /start command
func (b *botImpl) handlerStart(ctx *th.Context, update telego.Update) error {
	userID := update.Message.From.ID
	userSettings, err := b.getUserSettings(userID)
	if err != nil {
		log.Printf("Failed to get settings for user %d: %v", userID, err)
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Failed to get user settings."))
		return err
	}

	response := "👋 Welcome! Your current model is: `" + userSettings.ModelName + "`"
	_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), response).WithParseMode(telego.ModeMarkdown))
	return nil
}

// Handler for /listmodels command
func (b *botImpl) handlerListModels(ctx *th.Context, update telego.Update) error {
	userID := update.Message.From.ID
	models, err := b.geminiClient.ListModels(ctx)
	if err != nil {
		log.Printf("Failed to list models for user %d: %v", userID, err)
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Sorry, I couldn't retrieve the list of models."))
		return err
	}

	response := "✨ Available models:\n`" + strings.Join(models, "`\n`") + "`"
	_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), response).WithParseMode(telego.ModeMarkdown))
	return nil
}

// Handler for /setmodel command
func (b *botImpl) handlerSetModel(ctx *th.Context, update telego.Update) error {
	userID := update.Message.From.ID
	parts := strings.Fields(update.Message.Text)
	if len(parts) < 2 {
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "⚠️ Please specify a model name.\nUsage: `/setmodel model-name`").WithParseMode(telego.ModeMarkdown))
		return nil
	}
	modelName := parts[1]

	models, err := b.geminiClient.ListModels(ctx)
	if err != nil {
		log.Printf("Failed to list models for user %d: %v", userID, err)
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Sorry, I couldn't retrieve the list of models."))
		return err
	}
	if !slices.Contains(models, modelName) {
		log.Printf("Failed to list models for user %d: %v", userID, err)
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Sorry, Model not found."))
		return nil
	}

	userSettings, err := b.getUserSettings(userID)
	if err != nil {
		log.Printf("Failed to get settings for user %d: %v", userID, err)
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Failed to get user settings."))
		return err
	}
	if userSettings.ModelName == modelName {
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "⚠️ Model not changed (same)."))
		return nil
	}

	userSettings.ModelName = modelName
	if err = b.storage.SaveUserSettings(userID, userSettings); err != nil {
		log.Printf("Failed to save settings for user %d: %v", userID, err)
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Failed to save settings."))
		return err
	}

	log.Printf("Set model for user %d to %s", userID, modelName)
	response := "✅ Model successfully changed to `" + modelName + "`. A new chat session has been started."
	_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), response).WithParseMode(telego.ModeMarkdown))
	return nil
}

// Handler for /currentmodel command
func (b *botImpl) handlerCurrentModel(ctx *th.Context, update telego.Update) error {
	userID := update.Message.From.ID
	userSettings, err := b.getUserSettings(userID)
	if err != nil {
		log.Printf("Failed to get settings for user %d: %v", userID, err)
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Failed to get user settings."))
		return err
	}

	response := "✨ Your current model is: `" + userSettings.ModelName + "`"
	_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), response).WithParseMode(telego.ModeMarkdown))
	return nil
}

// Handler for /currentmodel command
func (b *botImpl) handlerDBSize(ctx *th.Context, update telego.Update) error {
	userID := update.Message.From.ID

	dbSize, err := b.storage.GetDBSize()
	if err != nil {
		return err
	}
	response := "✨ DB size is: `" + fmt.Sprintf("%.2f", dbSize) + "MB `"
	_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), response).WithParseMode(telego.ModeMarkdown))
	return nil
}

// Handler for all other messages
func (b *botImpl) handlerAnyMessage(ctx *th.Context, update telego.Update) error {
	userID := update.Message.From.ID
	text := update.Message.Text
	userSettings, err := b.getUserSettings(userID)
	if err != nil {
		log.Printf("Failed to get settings for user %d: %v", userID, err)
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Failed to get user settings."))
		return err
	}

	_ = ctx.Bot().SendChatAction(ctx, &telego.SendChatActionParams{
		ChatID: tu.ID(userID),
		Action: telego.ChatActionTyping,
	})

	response, err := b.geminiClient.GenerateContent(ctx, userSettings.History, userSettings.ModelName, text)
	if err != nil {
		log.Printf("Failed to get response from Gemini for user %d: %v", userID, err)
		errorMessage := "Sorry, I couldn't get a response. Please try again."
		if strings.Contains(err.Error(), "429:") {
			errorMessage = "❌ Gemini API rate limit exceeded. Please try again later."
		}
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), errorMessage))
		return err
	}

	userSettings.History.Messages = append(userSettings.History.Messages, storage.Message{Role: gemini.RoleUser, Text: text})
	userSettings.History.Messages = append(userSettings.History.Messages, storage.Message{Role: gemini.RoleModel, Text: response})
	if err = b.storage.SaveUserSettings(userID, userSettings); err != nil {
		log.Printf("Failed to save settings for user %d: %v", userID, err)
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Failed to save settings."))
		return err
	}

	key, err := b.storage.SaveResponse(userID, response)
	if err != nil {
		return err
	}

	err = b.SendLongMessage(ctx, tu.ID(userID), response)
	if err == nil {
		return nil
	}

	log.Printf("Failed to send message to user %d: %v", userID, err)
	_, err = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID),
		fmt.Sprintf("❌ Failed to send response message, message hash: %s", key)))
	return err
}
