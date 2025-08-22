package bot

import (
	"context"
	"errors"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/hashicorp/go-multierror"
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/vasyvasilie/gemini-chat-tg-bot/pkg/config"
	"github.com/vasyvasilie/gemini-chat-tg-bot/pkg/gemini"
	"github.com/vasyvasilie/gemini-chat-tg-bot/pkg/storage"
)

type Bot interface {
	Start()
	Stop()
}

var _ Bot = &botImpl{}

type botImpl struct {
	cfg          *config.Config
	storage      *storage.Storage
	tgBotHandler *th.BotHandler
	tgBotAPI     *tgbotapi.BotAPI
	geminiClient *gemini.Client
}

func (b *botImpl) SendLongMessage(ctx *th.Context, chatID telego.ChatID, text string) error {
	plainTextAfterMarkup, annotations := parseMarkupInternal(text)
	tgMessages, err := prepareTelegramMessages(plainTextAfterMarkup, annotations)
	if err != nil {
		return err
	}

	var mErr error
	for _, v := range tgMessages {
		msg := tgbotapi.NewMessage(chatID.ID, v.Text)
		for _, a := range v.Annotations {
			entity := tgbotapi.MessageEntity{
				Type:   llmSupportedPrefixes[a.Tag],
				Offset: a.UOffset,
				Length: a.Ulength,
			}
			msg.Entities = append(msg.Entities, entity)
		}
		_, err = b.tgBotAPI.Send(msg)
		if err != nil {
			mErr = multierror.Append(mErr, err)
		}
	}

	return mErr
}

func NewBot(ctx context.Context,
	cfg *config.Config,
	bolt *storage.Storage,
	tgBot *telego.Bot,
	geminiClient *gemini.Client,
) (Bot, error) {
	// Set bot commands for Telegram UI
	if err := tgBot.SetMyCommands(ctx, &telego.SetMyCommandsParams{
		Commands: []telego.BotCommand{
			{Command: "new", Description: "Start a new chat session"},
			{Command: "listmodels", Description: "Show available models"},
			{Command: "setmodel", Description: "Set a model (e.g. /setmodel model-name)"},
			{Command: "currentmodel", Description: "Show the currently selected model"},
			{Command: "dbsize", Description: "Show db size"},
		},
	}); err != nil {
		return nil, err
	}

	updates, err := tgBot.UpdatesViaLongPolling(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to get updates: %v", err)
	}

	tgBotHandler, err := th.NewBotHandler(tgBot, updates)
	if err != nil {
		return nil, err
	}

	tgBotAPI, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, err
	}

	bot := &botImpl{
		cfg:          cfg,
		storage:      bolt,
		tgBotHandler: tgBotHandler,
		tgBotAPI:     tgBotAPI,
		geminiClient: geminiClient,
	}
	bot.setupMiddlewares(ctx)
	bot.setupHandlers(ctx)

	return bot, nil
}

func (b *botImpl) Start() {
	b.tgBotHandler.Start()
}

func (b *botImpl) Stop() {
	b.tgBotHandler.Stop()
}

func (b *botImpl) getUserSettings(userID int64) (*storage.UserSettings, error) {
	settings, err := b.storage.GetUserSettings(userID)
	if err == nil {
		return settings, nil
	}

	if !errors.Is(err, storage.ErrUserNotFound) {
		return nil, err
	}

	settings = &storage.UserSettings{
		UserID:    userID,
		ModelName: b.cfg.DefaultModel,
	}

	if err = b.storage.SaveUserSettings(userID, settings); err != nil {
		return nil, err
	}

	return settings, nil
}
