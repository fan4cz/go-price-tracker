package worker

import (
	"context"
	"encoding/json"
	"go-price-tracker/internal/kafka"
	"go-price-tracker/internal/models"
	"go-price-tracker/internal/scraper"
	"go-price-tracker/internal/storage"
	"log/slog"
)

type ScraperWorker struct {
	consumer *kafka.Consumer
	scraper  scraper.Scraper
	repo     *storage.Repository
}

func NewScraperWorker(consumer *kafka.Consumer, scr scraper.Scraper, repo *storage.Repository) *ScraperWorker {
	return &ScraperWorker{
		consumer: consumer,
		scraper:  scr,
		repo:     repo,
	}
}

func (w *ScraperWorker) Start(ctx context.Context) {
	slog.Info("Scraper worker started")

	for {
		if ctx.Err() != nil {
			slog.Info("Scraper worker stoped")
			return
		}

		payload, err := w.consumer.ReadMessage(ctx)
		if err != nil {
			slog.Error("Ошибка чтения из Kafka", "error", err)
			continue
		}

		var job models.ScrapeJobEvent
		if err := json.Unmarshal(payload, &job); err != nil {
			slog.Error("Ошибка парсинга JSON задачи", "error", err, "payload", string(payload))
			continue
		}

		newPrice, err := w.scraper.FetchPrice(ctx, job.URL, job.Domain)
		if err != nil {
			slog.Error("Ошибка парсинга цены с сайта", "error", err, "url", job.URL)
			continue
		}

		err = w.repo.UpdateProductPrice(ctx, job.ProductID, newPrice)
		if err != nil {
			slog.Error("Ошибка обновления цены в БД", "error", err, "product_id", job.ProductID)
			continue
		}

		slog.Info("Цена успешно обновлена", "product_id", job.ProductID, "new_price", newPrice.StringFixed(2))

	}
}
