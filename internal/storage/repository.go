package storage

import (
	"context"
	"go-price-tracker/internal/models"

	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateUser(ctx context.Context, tgBotID int64) error {
	query := `
		INSERT INTO users (tg_bot_id)
		VALUES ($1)
		ON CONFLICT (tg_bot_id) DO NOTHING`

	_, err := r.db.ExecContext(ctx, query, tgBotID)
	return err
}

func (r *Repository) UpsertProduct(ctx context.Context, p models.Product) (int, error) {
	query := `
		INSERT INTO products (url, domain, current_price)
		VALUES ($1, $2, $3)
		ON CONFLICT (url) DO UPDATE
		SET current_price = EXCLUDED.current_price,
		last_checked_at = NOW()
		RETURNING id `

	var id int

	err := r.db.QueryRowContext(ctx, query, p.URL, p.Domain, p.CurrentPrice).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) AddSubscription(ctx context.Context, sub models.Subscription) error {
	query := `
		INSERT INTO subscriptions (user_id, product_id, target_price)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, product_id) DO UPDATE
		SET target_price = EXCLUDED.target_price`

	_, err := r.db.ExecContext(ctx, query, sub.UserID, sub.ProductID, sub.TargetPrice)
	return err
}

func (r *Repository) GetUserSubscriptions(ctx context.Context, userID int64) ([]models.UserSubscription, error) {
	query := `
		SELECT p.id, p.url, p.domain, p.current_price, p.last_checked_at, s.target_price
		FROM products p
		JOIN subscriptions s ON p.id = s.product_id
		WHERE s.user_id = $1`

	var subs []models.UserSubscription

	err := r.db.SelectContext(ctx, &subs, query, userID)
	if err != nil {
		return nil, err
	}
	return subs, nil
}
