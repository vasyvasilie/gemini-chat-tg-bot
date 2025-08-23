package bot

import (
	"context"
	"errors"
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

type UserSession struct {
	UserID         int64
	ModelName      string
	FavoriteModels []string
	History        []storage.Message
}

var (
	ErrModelNotFound = errors.New("model not found")
)

func (b *botImpl) getUserSessionWithErrorHandling(ctx *th.Context, userID int64) (*UserSession, error) {
	session, err := b.getUserSession(userID)
	if err != nil {
		log.Printf("Failed to get session for user %d: %v", userID, err)
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Failed to get user session."))
		return nil, err
	}
	return session, nil
}

func (b *botImpl) saveUserSessionWithErrorHandling(ctx *th.Context, session *UserSession, userID int64) error {
	settings := &storage.UserSettings{
		UserID:         session.UserID,
		ModelName:      session.ModelName,
		FavoriteModels: session.FavoriteModels,
		History:        storage.ConversationHistory{Messages: session.History},
	}
	if err := b.storage.SaveUserSettings(session.UserID, settings); err != nil {
		log.Printf("Failed to save session for user %d: %v", userID, err)
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Failed to save session."))
		return err
	}
	return nil
}

func (b *botImpl) sendErrorMessage(ctx *th.Context, userID int64, message string) {
	_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), message))
}

func (b *botImpl) sendSuccessMessage(ctx *th.Context, userID int64, message string) {
	_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), message))
}

func (b *botImpl) sendFormattedMessage(ctx *th.Context, userID int64, message string) {
	_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), message).WithParseMode(telego.ModeMarkdown))
}

func (b *botImpl) getModelsAndHandleErrors(ctx *th.Context, userID int64) ([]string, error) {
	models, err := b.geminiClient.ListModels(ctx)
	if err != nil {
		log.Printf("Failed to list models for user %d: %v", userID, err)
		b.sendErrorMessage(ctx, userID, "❌ Sorry, I couldn't retrieve the list of models.")
		return nil, err
	}
	return models, nil
}

func (b *botImpl) createModelKeyboard(models []string, prefix string) [][]telego.InlineKeyboardButton {
	var rows [][]telego.InlineKeyboardButton
	for _, model := range models {
		simpleModelName := strings.TrimPrefix(model, gemini.ModelPrefix)
		row := tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(simpleModelName).
				WithCallbackData(fmt.Sprintf("%s%s", prefix, simpleModelName)))
		rows = append(rows, row)
	}
	return rows
}

func (b *botImpl) getUserSession(userID int64) (*UserSession, error) {
	settings, err := b.storage.GetUserSettings(userID)
	if err == nil {
		return &UserSession{
			UserID:         settings.UserID,
			ModelName:      settings.ModelName,
			FavoriteModels: settings.FavoriteModels,
			History:        settings.History.Messages,
		}, nil
	}

	if !errors.Is(err, storage.ErrUserNotFound) {
		return nil, err
	}

	settings = &storage.UserSettings{
		UserID:    userID,
		ModelName: b.config.DefaultModel,
	}
	if err = b.storage.SaveUserSettings(userID, settings); err != nil {
		return nil, err
	}

	return &UserSession{
		UserID:    userID,
		ModelName: b.config.DefaultModel,
	}, nil
}

func (b *botImpl) setSessionModel(session *UserSession, modelName string) error {
	models, err := b.geminiClient.ListModels(context.Background())
	if err != nil {
		return err
	}

	if slices.Contains(models, modelName) {
		session.ModelName = modelName
		return nil
	}
	for _, model := range models {
		if model == modelName {
			session.ModelName = modelName
			return nil
		}
	}

	return ErrModelNotFound
}
