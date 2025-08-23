package bot

import (
	"context"
	"fmt"
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
	config       *config.Config
	storage      *storage.Storage
	tgBotHandler *th.BotHandler
	tgBotAPI     *tgbotapi.BotAPI
	geminiClient *gemini.Client
}

func (b *botImpl) SendLongMessage(ctx *th.Context, chatID telego.ChatID, text string) error {
	plainTextAfterMarkup, annotations := parseMarkupInternal(text)
	tgMessages, err := prepareTelegramMessages(plainTextAfterMarkup, annotations)
	if err != nil {
		return fmt.Errorf("Cannot prepare telegram messages: %w", err)
	}

	var multiErr error
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
			multiErr = multierror.Append(multiErr, err)
		}
	}

	return multiErr
}

func NewBot(ctx context.Context,
	config *config.Config,
	bolt *storage.Storage,
	tgBot *telego.Bot,
	geminiClient *gemini.Client,
) (Bot, error) {
	// Set bot commands for Telegram UI
	if err := tgBot.SetMyCommands(ctx, &telego.SetMyCommandsParams{Commands: botCommands}); err != nil {
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

	tgBotAPI, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		return nil, err
	}

	bot := &botImpl{
		config:       config,
		storage:      bolt,
		tgBotHandler: tgBotHandler,
		tgBotAPI:     tgBotAPI,
		geminiClient: geminiClient,
	}
	bot.setupMiddlewares()
	bot.setupHandlers()

	return bot, nil
}

func (b *botImpl) Start() {
	b.tgBotHandler.Start()
}

func (b *botImpl) Stop() {
	b.tgBotHandler.Stop()
}
