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

// user friendly ordered
var botCommands = []telego.BotCommand{
	{Command: "new", Description: "Start a new chat session"},
	{Command: "currentmodel", Description: "Show the currently selected model"},
	{Command: "selectmodel", Description: "Select model from favorites"},
	{Command: "dbsize", Description: "Show db size"},
	{Command: "listmodels", Description: "Show available models plain text"},
	{Command: "setmodel", Description: "Set a model (e.g. /setmodel model-name)"},
	{Command: "addmodeltofavorites", Description: "Add model to favorites"},
	{Command: "clearfavorites", Description: "Clear favorites"},
}

func (b *botImpl) setupHandlers(ctx context.Context) {
	// handlers
	b.tgBotHandler.Handle(b.handlerNew, th.CommandEqual("new"))
	b.tgBotHandler.Handle(b.handlerStart, th.CommandEqual("start"))
	b.tgBotHandler.Handle(b.handlerListModels, th.CommandEqual("listmodels"))
	b.tgBotHandler.Handle(b.handlerSetModel, th.CommandEqual("setmodel"))
	b.tgBotHandler.Handle(b.handlerCurrentModel, th.CommandEqual("currentmodel"))
	b.tgBotHandler.Handle(b.handlerDBSize, th.CommandEqual("dbsize"))
	b.tgBotHandler.Handle(b.handlerAddModelToFavorites, th.CommandEqual("addmodeltofavorites"))
	b.tgBotHandler.Handle(b.handlerSelectModel, th.CommandEqual("selectmodel"))
	b.tgBotHandler.Handle(b.handlerClearFavorites, th.CommandEqual("clearfavorites"))
	b.tgBotHandler.Handle(b.handlerAnyMessage, th.AnyMessage())

	// callbacks
	b.tgBotHandler.HandleCallbackQuery(b.callbackAddModelToFavorites, th.CallbackDataPrefix(prefixAddModelToFavorites))
	b.tgBotHandler.HandleCallbackQuery(b.callbackSetModelFromFavorites, th.CallbackDataPrefix(prefixSetModelFromFavorites))
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

	_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID),
		fmt.Sprintf("👋 Welcome! Your current model is: `%s`", userSettings.ModelName)).
		WithParseMode(telego.ModeMarkdown))
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

	_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID),
		fmt.Sprintf("✨ Available models:\n`%s`", strings.Join(models, "`\n`"))).
		WithParseMode(telego.ModeMarkdown))
	return nil
}

// Handler for /setmodel command
func (b *botImpl) handlerSetModel(ctx *th.Context, update telego.Update) error {
	userID := update.Message.From.ID
	parts := strings.Fields(update.Message.Text)
	if len(parts) < 2 {
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID),
			"⚠️ Please specify a model name.\nUsage: `/setmodel model-name`").
			WithParseMode(telego.ModeMarkdown))
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

	_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID),
		fmt.Sprintf("✅ Model successfully changed to `%s`. A new chat session has been started.", modelName)).
		WithParseMode(telego.ModeMarkdown))
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

	_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID),
		fmt.Sprintf("✨ Your current model is: `%s`", userSettings.ModelName)).
		WithParseMode(telego.ModeMarkdown))
	return nil
}

// Handler for /currentmodel command
func (b *botImpl) handlerDBSize(ctx *th.Context, update telego.Update) error {
	userID := update.Message.From.ID

	dbSize, err := b.storage.GetDBSize()
	if err != nil {
		return err
	}

	_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID),
		fmt.Sprintf("✨ DB size is: `%.2fMB`", dbSize)).
		WithParseMode(telego.ModeMarkdown))
	return nil
}

// Handler for /addmodeltofavorites command
func (b *botImpl) handlerAddModelToFavorites(ctx *th.Context, update telego.Update) error {
	userID := update.Message.From.ID

	models, err := b.geminiClient.ListModels(ctx)
	if err != nil {
		log.Printf("Failed to list models for user %d: %v", userID, err)
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Sorry, I couldn't retrieve the list of models."))
		return err
	}

	var rows [][]telego.InlineKeyboardButton
	for _, model := range models {
		simpleModelName := strings.TrimPrefix(model, gemini.ModelPrefix)
		row := tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(simpleModelName).
				WithCallbackData(fmt.Sprintf("%s%s", prefixAddModelToFavorites, simpleModelName)))
		rows = append(rows, row)
	}

	_, err = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "💟 Add model to favorite").
		WithReplyMarkup(tu.InlineKeyboard(rows...)))
	return err
}

// Handler for /selectmodel command
func (b *botImpl) handlerSelectModel(ctx *th.Context, update telego.Update) error {
	userID := update.Message.From.ID
	userSettings, err := b.getUserSettings(userID)
	if err != nil {
		log.Printf("Failed to get settings for user %d: %v", userID, err)
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Failed to get user settings."))
		return err
	}

	if len(userSettings.FavoriteModels) == 0 {
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❎ Empty favorites."))
		return nil
	}

	var rows [][]telego.InlineKeyboardButton
	for _, model := range userSettings.FavoriteModels {
		rows = append(rows, tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(strings.TrimPrefix(model, gemini.ModelPrefix)).
				WithCallbackData(fmt.Sprintf("%s%s", prefixSetModelFromFavorites, model))))
	}

	_, err = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "💟 Set model from favorites.").
		WithReplyMarkup(tu.InlineKeyboard(rows...)))
	return err
}

// Handler for /new command
func (b *botImpl) handlerClearFavorites(ctx *th.Context, update telego.Update) error {
	userID := update.Message.From.ID
	userSettings, err := b.getUserSettings(userID)
	if err != nil {
		log.Printf("Failed to get settings for user %d: %v", userID, err)
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Failed to get user settings."))
		return err
	}

	userSettings.FavoriteModels = nil
	if err = b.storage.SaveUserSettings(userID, userSettings); err != nil {
		log.Printf("Failed to save settings for user %d: %v", userID, err)
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Failed to save settings."))
		return err
	}

	log.Printf("Started new session for user %d with model %s", userID, userSettings.ModelName)
	_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "✅ Deleted all favorite models."))
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
