package bot

import (
	"log"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

const (
	unauthorizedMsg string = "Unauthorized access from user ID: %d"
)

func (b *botImpl) setupMiddlewares() {
	b.tgBotHandler.Use(b.middlewareAccess)
}

func (b *botImpl) middlewareAccess(ctx *th.Context, update telego.Update) error {
	if update.Message == nil {
		return ctx.Next(update)
	}

	userID := update.Message.From.ID
	if _, ok := b.config.AllowedUsers[userID]; !ok {
		log.Printf(unauthorizedMsg, userID)
		return nil
	}

	return ctx.Next(update)
}
