package bot

import (
	"fmt"
	"log"
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

func (b *botImpl) callbackAddModelToFavorites(ctx *th.Context, query telego.CallbackQuery) error {
	chatID := query.Message.GetChat().ChatID()
	userID := chatID.ID

	if err := ctx.Bot().AnswerCallbackQuery(ctx, tu.CallbackQuery(query.ID)); err != nil {
		log.Printf("Failed to answer callback: %v", err)
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Failed to answer callback."))
		return err
	}

	if _, err := ctx.Bot().EditMessageReplyMarkup(ctx, tu.EditMessageReplayMarkup(
		chatID,
		query.Message.GetMessageID(),
		nil,
	)); err != nil {
		log.Printf("Failed remove inline keyboard: %v", err)
		return err
	}

	fullModelName := fmt.Sprintf("%s%s",
		gemini.ModelPrefix,
		strings.TrimPrefix(query.Data, prefixAddModelToFavorites),
	)

	userSettings, err := b.getUserSettings(userID)
	if err != nil {
		log.Printf("Failed to get settings for user %d: %v", userID, err)
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Failed to get user settings."))
		return err
	}

	for _, model := range userSettings.FavoriteModels {
		if model == fullModelName {
			_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID),
				fmt.Sprintf("❎ Model `%s` was in favorites already.", fullModelName)).WithParseMode(telego.ModeMarkdown))
			return nil
		}
	}

	userSettings.FavoriteModels = append(userSettings.FavoriteModels, fullModelName)
	sort.Strings(userSettings.FavoriteModels)
	if err = b.storage.SaveUserSettings(userID, userSettings); err != nil {
		log.Printf("Failed to save settings for user %d: %v", userID, err)
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Failed to save settings."))
		return err
	}

	_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID),
		fmt.Sprintf("✅ Model `%s` added to favorites.", fullModelName)).WithParseMode(telego.ModeMarkdown))
	return nil
}

func (b *botImpl) callbackSetModelFromFavorites(ctx *th.Context, query telego.CallbackQuery) error {
	chatID := query.Message.GetChat().ChatID()
	userID := chatID.ID

	if err := ctx.Bot().AnswerCallbackQuery(ctx, tu.CallbackQuery(query.ID)); err != nil {
		log.Printf("Failed to answer callback: %v", err)
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Failed to answer callback."))
		return err
	}

	if _, err := ctx.Bot().EditMessageReplyMarkup(ctx, tu.EditMessageReplayMarkup(
		chatID,
		query.Message.GetMessageID(),
		nil,
	)); err != nil {
		log.Printf("Failed remove inline keyboard: %v", err)
		return err
	}

	fullModelName := strings.TrimPrefix(query.Data, prefixSetModelFromFavorites)
	userSettings, err := b.getUserSettings(userID)
	if err != nil {
		log.Printf("Failed to get settings for user %d: %v", userID, err)
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Failed to get user settings."))
		return err
	}

	userSettings.ModelName = fullModelName
	if err = b.storage.SaveUserSettings(userID, userSettings); err != nil {
		log.Printf("Failed to save settings for user %d: %v", userID, err)
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID), "❌ Failed to save settings."))
		return err
	}

	_, _ = ctx.Bot().SendMessage(ctx, tu.Message(tu.ID(userID),
		fmt.Sprintf("✨ Your current model is: `%s`", fullModelName)).WithParseMode(telego.ModeMarkdown))
	return nil
}
