package scheduler

import (
	"context"
	"go-price-tracker/internal/storage"
	"log/slog"
	"time"
)

type Config struct {
	TickInterval time.Duration
	OutdatedAge  time.Duration
	BatchSize    int
}

type Scheduler struct {
	repo *storage.Repository
	cfg  Config
}

func NewScheduler(repo *storage.Repository, cfg Config) *Scheduler {
	return &Scheduler{
		repo: repo,
		cfg:  cfg,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	slog.Info("Планировщик запущен",
		"tick_interval", s.cfg.TickInterval,
		"outdated_age", s.cfg.OutdatedAge,
	)

	ticker := time.NewTicker(s.cfg.TickInterval)
	defer ticker.Stop()

	s.process(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("Остановка планировщика...")
			return
		case <-ticker.C:
			s.process(ctx)
		}
	}
}

func (s *Scheduler) process(ctx context.Context) {
	products, err := s.repo.GetOutdatedProducts(ctx, s.cfg.OutdatedAge, s.cfg.BatchSize)
	if err != nil {
		slog.Error("Не удалось получить устаревшие товары", "error", err)
		return
	}

	if len(products) == 0 {
		slog.Debug("Нет товаров, требующих проверки цен")
		return
	}

	slog.Info("Найдены товары для обновления", "count", len(products))

	ids := make([]int, 0, len(products))
	for _, p := range products {
		ids = append(ids, p.ID)

		slog.Debug("Сформирована задача на парсинг", "product_id", p.ID, "url", p.URL)
	}

	if err := s.repo.MarkProductsAsChecked(ctx, ids); err != nil {
		slog.Error("Ошибка обновления статуса проверки товаров", "error", err)
	}
}
