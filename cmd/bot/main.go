package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/sergey/GoogleTaskBot/internal/config"
	"github.com/sergey/GoogleTaskBot/internal/httpclient"
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

	if err := os.MkdirAll(filepath.Dir(cfg.NotifyDBPath), 0o755); err != nil {
		log.Fatalf("notify db dir: %v", err)
	}
	store, err := notify.Open(cfg.NotifyDBPath)
	if err != nil {
		log.Fatalf("notify store: %v", err)
	}
	defer store.Close()

	httpClient, err := httpclient.New(cfg.HTTPProxy)
	if err != nil {
		log.Fatalf("http client: %v", err)
	}
	if cfg.HTTPProxy != "" {
		log.Println("using HTTP proxy for Telegram and Google API")
	}

	client, err := tasks.NewClient(ctx, cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRefreshToken, httpClient)
	if err != nil {
		log.Fatalf("google tasks: %v", err)
	}
	if err := client.Ping(ctx); err != nil {
		log.Fatalf("google tasks auth check: %v", err)
	}
	log.Println("google tasks API connected")

	tg, err := telegram.New(cfg, client, httpClient, store)
	if err != nil {
		log.Fatalf("telegram: %v", err)
	}

	notifier := scheduler.New(cfg, client, tg.TeleBot(), store)
	notifier.Start(ctx)
	notifier.SendStartup()

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		cancel()
		notifier.Stop()
		tg.TeleBot().Stop()
	}()

	tg.Run()
}
