package bot

import (
	"context"
	"log"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

func (b *botImpl) setupMiddlewares(ctx context.Context) {
	b.tgBotHandler.Use(b.middlewareAccess)
}

func (b *botImpl) middlewareAccess(ctx *th.Context, update telego.Update) error {
	var userID int64
	if update.Message == nil {
		return ctx.Next(update)
	}

	userID = update.Message.From.ID
	if _, ok := b.cfg.AllowedUsers[userID]; !ok {
		log.Printf("Unauthorized access from user ID: %d", userID)
		return nil
	}
	return ctx.Next(update)
}
