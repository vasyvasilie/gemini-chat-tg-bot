package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/mymmrac/telego"

	botPkg "github.com/vasyvasilie/gemini-chat-tg-bot/pkg/bot"
	"github.com/vasyvasilie/gemini-chat-tg-bot/pkg/config"
	"github.com/vasyvasilie/gemini-chat-tg-bot/pkg/gemini"
	"github.com/vasyvasilie/gemini-chat-tg-bot/pkg/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	bolt, err := storage.NewStorage(cfg.StoragePath)
	if err != nil {
		log.Fatal(err)
	}

	tgBot, err := telego.NewBot(cfg.BotToken, telego.WithDefaultLogger(cfg.Debug, true))
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	geminiClient, err := gemini.NewClient(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}

	bot, err := botPkg.NewBot(ctx, cfg, bolt, tgBot, geminiClient)
	if err != nil {
		log.Fatalf("Failed to create bot handler: %v", err)
	}
	defer bot.Stop()

	log.Println("Bot starting...")
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		bot.Start()
	}()

	<-ctx.Done()
	log.Println("Stopping bot...")
	bot.Stop()
	wg.Wait()
	log.Println("Bot stopped.")
}
