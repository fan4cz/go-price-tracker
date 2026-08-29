package main

import (
	"context"
	"go-price-tracker/internal/config"
	"go-price-tracker/internal/kafka"
	"go-price-tracker/internal/scheduler"
	"go-price-tracker/internal/storage"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	slog.Info("Запуск планировзика", "env", cfg.Env)

	db, err := sqlx.Connect("pgx", cfg.Database.URL)
	if err != nil {
		slog.Error("Ошибка подключения к Postgres в Scheduler", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	repo := storage.NewRepository(db)

	producer := kafka.NewProducer(cfg.Kafka.Brokers)
	defer producer.Close()

	sch := scheduler.NewScheduler(repo, producer, scheduler.Config{
		TickInterval: 30 * time.Second,
		OutdatedAge:  1 * time.Hour,
		BatchSize:    50,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sch.Start(ctx)
}
