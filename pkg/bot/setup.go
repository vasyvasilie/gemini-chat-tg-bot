package bot

import (
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
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

func (b *botImpl) setupHandlers() {
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
