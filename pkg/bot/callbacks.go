package bot

import (
	"fmt"
	"log"
	"slices"
	"sort"
	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/vasyvasilie/gemini-chat-tg-bot/pkg/gemini"
)

const (
	prefixAddModelToFavorites   string = "v1_add_"
	prefixSetModelFromFavorites string = "v1_setmodelfromfavorites_"
)

func (b *botImpl) setupCallbackQuery(ctx *th.Context, query telego.CallbackQuery, userID int64) error {
	if err := ctx.Bot().AnswerCallbackQuery(ctx, tu.CallbackQuery(query.ID)); err != nil {
		log.Printf("Failed to answer callback: %v", err)
		b.sendErrorMessage(ctx, userID, "❌ Failed to answer callback.")
		return err
	}

	chatID := query.Message.GetChat().ChatID()
	if _, err := ctx.Bot().EditMessageReplyMarkup(ctx, tu.EditMessageReplayMarkup(
		chatID,
		query.Message.GetMessageID(),
		nil,
	)); err != nil {
		log.Printf("Failed remove inline keyboard: %v", err)
		return err
	}
	return nil
}

func (b *botImpl) callbackAddModelToFavorites(ctx *th.Context, query telego.CallbackQuery) error {
	chatID := query.Message.GetChat().ChatID()
	userID := chatID.ID
	session, err := b.getUserSessionWithErrorHandling(ctx, userID)
	if err != nil {
		return err
	}

	if err := b.setupCallbackQuery(ctx, query, userID); err != nil {
		return err
	}

	fullModelName := fmt.Sprintf("%s%s",
		gemini.ModelPrefix,
		strings.TrimPrefix(query.Data, prefixAddModelToFavorites),
	)

	if slices.Contains(session.FavoriteModels, fullModelName) {
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID),
			fmt.Sprintf("❎ Model `%s` was in favorites already.", fullModelName)).WithParseMode(telego.ModeMarkdown))
		return nil
	}

	session.FavoriteModels = append(session.FavoriteModels, fullModelName)
	sort.Strings(session.FavoriteModels)
	if err = b.saveUserSessionWithErrorHandling(ctx, session, userID); err != nil {
		return err
	}

	b.sendFormattedMessage(ctx, userID, fmt.Sprintf("✅ Model `%s` added to favorites.", fullModelName))
	return nil
}

func (b *botImpl) callbackSetModelFromFavorites(ctx *th.Context, query telego.CallbackQuery) error {
	chatID := query.Message.GetChat().ChatID()
	userID := chatID.ID

	if err := b.setupCallbackQuery(ctx, query, userID); err != nil {
		return err
	}

	session, err := b.getUserSessionWithErrorHandling(ctx, userID)
	if err != nil {
		return err
	}

	fullModelName := strings.TrimPrefix(query.Data, prefixSetModelFromFavorites)
	if err = b.setSessionModel(session, fullModelName); err != nil {
		log.Printf("Failed to set model for user %d: %v", userID, err)
		b.sendErrorMessage(ctx, userID, "❌ Failed to set model.")
		return err
	}

	if err = b.saveUserSessionWithErrorHandling(ctx, session, userID); err != nil {
		return err
	}

	b.sendFormattedMessage(ctx, userID, fmt.Sprintf("✨ Your current model is: `%s`", fullModelName))
	return nil
}
