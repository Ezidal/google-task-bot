package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sergey/GoogleTaskBot/internal/config"
	"github.com/sergey/GoogleTaskBot/internal/notify"
	"github.com/sergey/GoogleTaskBot/internal/scheduler"
	"github.com/sergey/GoogleTaskBot/internal/tasks"
	"github.com/sergey/GoogleTaskBot/internal/telegram"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := tasks.NewClient(ctx, cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRefreshToken)
	if err != nil {
		log.Fatalf("google tasks: %v", err)
	}

	tg, err := telegram.New(cfg, client)
	if err != nil {
		log.Fatalf("telegram: %v", err)
	}

	store := notify.NewStore()
	notifier := scheduler.New(cfg, client, tg.TeleBot(), store)
	notifier.Start(ctx)

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		cancel()
		tg.TeleBot().Stop()
	}()

	tg.Run()
}
