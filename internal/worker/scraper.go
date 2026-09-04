package worker

import (
	"context"
	"encoding/json"
	"go-price-tracker/internal/kafka"
	"go-price-tracker/internal/models"
	"go-price-tracker/internal/scraper"
	"go-price-tracker/internal/storage"
	"log/slog"
	"strconv"
)

type ScraperWorker struct {
	consumer *kafka.Consumer
	producer *kafka.Producer
	scraper  scraper.Scraper
	repo     *storage.Repository
}

func NewScraperWorker(consumer *kafka.Consumer, producer *kafka.Producer, scr scraper.Scraper, repo *storage.Repository) *ScraperWorker {
	return &ScraperWorker{
		consumer: consumer,
		producer: producer,
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

		subs, err := w.repo.GetSubscriptionsToAlert(ctx, job.ProductID, newPrice)
		if err != nil || len(subs) == 0 {
			continue
		}

		for _, sub := range subs {
			alert := models.AlertEvent{
				UserID:      sub.UserID,
				URL:         job.URL,
				NewPrice:    newPrice.StringFixed(2),
				TargetPrice: sub.TargetPrice.StringFixed(2),
			}

			alertBytes, _ := json.Marshal(alert)

			key := []byte(strconv.FormatInt(sub.UserID, 10))

			err = w.producer.PublishMessage(ctx, key, alertBytes)
			if err == nil {
				slog.Info("Уведомление отправлено в очередь", "user", sub.UserID)
			} else {
				slog.Error("Не удалось отправить алерт", "error", err, "user_id", sub.UserID)
			}
		}

	}
}
