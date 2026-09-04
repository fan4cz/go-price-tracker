package storage

import (
	"context"
	"fmt"
	"go-price-tracker/internal/models"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
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

func (r *Repository) GetOutdatedProducts(ctx context.Context, interval time.Duration, limit int) ([]models.Product, error) {
	query := `
			SELECT id, url, domain, current_price, last_checked_at
			FROM products
			WHERE last_checked_at IS NULL
			   OR last_checked_at < NOW() - $1::interval
			ORDER BY last_checked_at ASC NULLS FIRST
			LIMIT $2`

	intervalParam := fmt.Sprintf("%d seconds", int(interval.Seconds()))

	var products []models.Product
	err := r.db.SelectContext(ctx, &products, query, intervalParam, limit)
	if err != nil {
		return nil, fmt.Errorf("ошибка выборки устаревших товаров: %w", err)
	}

	return products, nil
}

func (r *Repository) MarkProductsAsChecked(ctx context.Context, productIDs []int) error {
	if len(productIDs) == 0 {
		return nil
	}

	query, args, err := sqlx.In(`
		UPDATE products
		SET last_checked_at = NOW()
		WHERE id IN (?)
	`, productIDs)
	if err != nil {
		return fmt.Errorf("ошибка формирования IN-запроса: %w", err)
	}

	query = r.db.Rebind(query)
	_, err = r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *Repository) UpdateProductPrice(ctx context.Context, productID int, newPrice decimal.Decimal) error {
	query := `
		Update products
		Set current_price = $1
		Where id = $2
	`
	_, err := r.db.ExecContext(ctx, query, productID, newPrice)
	return err
}

// get all users with product witch lower then target price
func (r *Repository) GetSubscriptionsToAlert(ctx context.Context, productID int, newPrice decimal.Decimal) ([]models.Subscription, error) {
	query := `
		SELECT user_id, product_id, target_price 
		FROM subscriptions 
		WHERE product_id = $1 AND target_price >= $2
	`
	var subs []models.Subscription
	err := r.db.SelectContext(ctx, &subs, query, productID, newPrice)
	return subs, err
}