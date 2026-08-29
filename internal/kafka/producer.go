package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"go-price-tracker/internal/models"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
)

const TopicScrapeJobs = "scrape_jobs"

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        TopicScrapeJobs,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafka.RequireOne,
	}

	return &Producer{writer: writer}
}

func (p *Producer) PublishScrapeJobs(ctx context.Context, jobs []models.ScrapeJobEvent) error {
	messages := make([]kafka.Message, 0, len(jobs))

	for _, job := range jobs {
		payload, err := json.Marshal(job)
		if err != nil {
			return fmt.Errorf("ошибка сериализации задачи product_id=%d: %w", job.ProductID, err)
		}

		messages = append(messages, kafka.Message{
			Key:   []byte(job.Domain),
			Value: payload,
		})
	}

	if err := p.writer.WriteMessages(ctx, messages...); err != nil {
		return fmt.Errorf("ошибка отправки сообщений в Kafka: %w", err)
	}

	slog.Info("Задачи успешно отправлены в Kafka", "count", len(messages), "topic", TopicScrapeJobs)
	return nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
