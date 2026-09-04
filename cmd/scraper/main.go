package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"go-price-tracker/internal/config"
	"go-price-tracker/internal/kafka"
	"go-price-tracker/internal/scraper"
	"go-price-tracker/internal/storage"
	"go-price-tracker/internal/worker"
)

func main() {
	cfg := config.MustLoad()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	db, err := sqlx.Connect("pgx", cfg.Database.URL)
	if err != nil {
		slog.Error("Ошибка подключения к БД в Scraper Worker", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	repo := storage.NewRepository(db)

	webScraper := scraper.NewScraper()

	consumer := kafka.NewConsumer(cfg.Kafka.Brokers, "scraper_workers_group", kafka.TopicScrapeJobs)
	defer consumer.Close()

	scraperWorker := worker.NewScraperWorker(consumer, webScraper, repo)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	scraperWorker.Start(ctx)
}
