package main

import (
	"context"
	"go-price-tracker/internal/bot"
	"go-price-tracker/internal/config"
	"go-price-tracker/internal/kafka"
	"go-price-tracker/internal/scraper"
	"go-price-tracker/internal/service"
	"go-price-tracker/internal/storage"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

func main() {
	cfg := config.MustLoad()

	var logger *slog.Logger
	if cfg.Env == "local" {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	} else {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	slog.SetDefault(logger)

	slog.Info("Запуск приложения", "env", cfg.Env)

	if err := storage.RunMigrations(cfg.Database.URL); err != nil {
		slog.Error("Ошибка применения миграций", "error", err)
		os.Exit(1)
	}

	db, err := sqlx.Connect("pgx", cfg.Database.URL)
	if err != nil {
		slog.Error("Ошибка подключения к Postgres", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	repo := storage.NewRepository(db)
	webScraper := scraper.NewScraper()
	trackerService := service.NewTrackerService(repo, webScraper)
	alertConsumer := kafka.NewConsumer(cfg.Kafka.Brokers, "bot_alerts_group", "alerts")
	defer alertConsumer.Close()

	tgBot, err := bot.NewBot(cfg.Telegram.Token, trackerService)
	if err != nil {
		slog.Error("Не удалось инициализировать бота", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go tgBot.StartAlertsListener(ctx)

	if err := tgBot.Start(ctx); err != nil {
		slog.Error("Критическая ошибка работы бота", "error", err)
	}

}
