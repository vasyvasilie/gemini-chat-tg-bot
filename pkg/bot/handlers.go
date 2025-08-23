package bot

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/vasyvasilie/gemini-chat-tg-bot/pkg/gemini"
	"github.com/vasyvasilie/gemini-chat-tg-bot/pkg/storage"
)

// Handler for /new command
func (b *botImpl) handlerNew(ctx *th.Context, update telego.Update) error {
	userID := update.Message.From.ID
	session, err := b.getUserSessionWithErrorHandling(ctx, userID)
	if err != nil {
		return err
	}

	session.History = []storage.Message{}
	if err = b.saveUserSessionWithErrorHandling(ctx, session, userID); err != nil {
		return err
	}

	log.Printf("Started new session for user %d with model %s", userID, session.ModelName)
	b.sendSuccessMessage(ctx, userID, "✅ New chat session started. Your history has been cleared.")
	return nil
}

// Handler for /start command
func (b *botImpl) handlerStart(ctx *th.Context, update telego.Update) error {
	userID := update.Message.From.ID
	session, err := b.getUserSessionWithErrorHandling(ctx, userID)
	if err != nil {
		return err
	}

	b.sendFormattedMessage(ctx, userID, fmt.Sprintf("👋 Welcome! Your current model is: `%s`", session.ModelName))
	return nil
}

// Handler for /listmodels command
func (b *botImpl) handlerListModels(ctx *th.Context, update telego.Update) error {
	userID := update.Message.From.ID
	models, err := b.getModelsAndHandleErrors(ctx, userID)
	if err != nil {
		return err
	}

	b.sendFormattedMessage(ctx, userID,
		fmt.Sprintf("✨ Available models:\n`%s`", strings.Join(models, "`\n`")))
	return nil
}

// Handler for /setmodel command
func (b *botImpl) handlerSetModel(ctx *th.Context, update telego.Update) error {
	userID := update.Message.From.ID
	session, err := b.getUserSessionWithErrorHandling(ctx, userID)
	if err != nil {
		return err
	}

	parts := strings.Fields(update.Message.Text)
	if len(parts) < 2 {
		b.sendFormattedMessage(ctx, userID,
			"⚠️ Please specify a model name.\nUsage: `/setmodel model-name`")
		return nil
	}

	modelName := parts[1]
	if session.ModelName == modelName {
		b.sendSuccessMessage(ctx, userID, "⚠️ Model not changed (same).")
		return nil
	}

	if err = b.setSessionModel(session, modelName); err != nil {
		log.Printf("Failed to set model for user %d: %v", userID, err)
		b.sendErrorMessage(ctx, userID, "❌ Failed to set model.")
		return nil
	}

	if err = b.saveUserSessionWithErrorHandling(ctx, session, userID); err != nil {
		return err
	}

	log.Printf("Set model for user %d to %s", userID, modelName)
	b.sendFormattedMessage(ctx, userID,
		fmt.Sprintf("✅ Model successfully changed to `%s`. A new chat session has been started.", modelName))
	return nil
}

// Handler for /currentmodel command
func (b *botImpl) handlerCurrentModel(ctx *th.Context, update telego.Update) error {
	userID := update.Message.From.ID
	session, err := b.getUserSessionWithErrorHandling(ctx, userID)
	if err != nil {
		return err
	}

	b.sendFormattedMessage(ctx, userID, fmt.Sprintf("✨ Your current model is: `%s`", session.ModelName))
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
	models, err := b.getModelsAndHandleErrors(ctx, userID)
	if err != nil {
		return err
	}

	rows := b.createModelKeyboard(models, prefixAddModelToFavorites)
	_, err = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "💟 Add model to favorite").
		WithReplyMarkup(tu.InlineKeyboard(rows...)))
	return err
}

// Handler for /selectmodel command
func (b *botImpl) handlerSelectModel(ctx *th.Context, update telego.Update) error {
	userID := update.Message.From.ID
	session, err := b.getUserSessionWithErrorHandling(ctx, userID)
	if err != nil {
		return err
	}

	if len(session.FavoriteModels) == 0 {
		b.sendErrorMessage(ctx, userID, "❎ Empty favorites.")
		return nil
	}

	var rows [][]telego.InlineKeyboardButton
	for _, model := range session.FavoriteModels {
		rows = append(rows, tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(strings.TrimPrefix(model, gemini.ModelPrefix)).
				WithCallbackData(fmt.Sprintf("%s%s", prefixSetModelFromFavorites, model))))
	}

	_, err = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "💟 Set model from favorites.").
		WithReplyMarkup(tu.InlineKeyboard(rows...)))
	return err
}

// Handler for /clearfavorites command
func (b *botImpl) handlerClearFavorites(ctx *th.Context, update telego.Update) error {
	userID := update.Message.From.ID
	session, err := b.getUserSessionWithErrorHandling(ctx, userID)
	if err != nil {
		return err
	}

	session.FavoriteModels = []string{}
	if err = b.saveUserSessionWithErrorHandling(ctx, session, userID); err != nil {
		return err
	}

	log.Printf("Cleared favorites for user %d", userID)
	b.sendSuccessMessage(ctx, userID, "✅ Deleted all favorite models.")
	return nil
}

// Handler for all other messages
func (b *botImpl) handlerAnyMessage(ctx *th.Context, update telego.Update) error {
	userID := update.Message.From.ID
	session, err := b.getUserSessionWithErrorHandling(ctx, userID)
	if err != nil {
		return err
	}

	_ = ctx.Bot().SendChatAction(ctx, &telego.SendChatActionParams{
		ChatID: tu.ID(userID),
		Action: telego.ChatActionTyping,
	})

	text := update.Message.Text
	history := storage.ConversationHistory{Messages: session.History}
	response, err := b.geminiClient.GenerateContent(ctx, history, session.ModelName, text)
	if err != nil {
		log.Printf("Failed to get response from Gemini for user %d: %v", userID, err)

		key, logErr := b.storage.LogGeminiError(userID, session.ModelName, text, err.Error(), session.History)
		if logErr != nil {
			log.Printf("Cannot save error for user %d: %v", userID, logErr)
		}

		if errors.Is(err, gemini.GeminiTooManyRequestError) {
			b.sendErrorMessage(ctx, userID, "❌ Gemini API rate limit exceeded. Please try again later.")
			return nil
		}

		if errors.Is(err, gemini.GeminiEmptyAnswer) {
			b.sendErrorMessage(ctx, userID, "❌ Gemini API answered with empty text.")
			return nil
		}

		b.sendErrorMessage(ctx, userID, fmt.Sprintf("❌ Gemini API error, saved: %s", key))
		return err
	}

	session.History = append(session.History, storage.Message{Role: gemini.RoleUser, Text: text})
	session.History = append(session.History, storage.Message{Role: gemini.RoleModel, Text: response})
	if err = b.saveUserSessionWithErrorHandling(ctx, session, userID); err != nil {
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
		fmt.Sprintf("❌ Failed to send response message, response hash: %s", key)))
	return err
}
