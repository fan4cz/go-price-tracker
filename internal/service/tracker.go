package service

import (
	"context"
	"go-price-tracker/internal/models"
	"go-price-tracker/internal/storage"
	"net/url"
	"strings"

	"github.com/shopspring/decimal"
)

type TrackerService interface {
	TrackProduct(ctx context.Context, userID int64, rawURL string, targetPriceStr string) error
	GetUserList(ctx context.Context, userID int64) ([]models.UserSubscription, error)
}

type trackerService struct {
	repo *storage.Repository
}

func NewTrackerService(repo *storage.Repository) TrackerService {
	return &trackerService{
		repo: repo,
	}
}

func (s trackerService) TrackProduct(ctx context.Context, userID int64, rawURL string, targetPriceStr string) error {
	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return err
	}

	domain := strings.TrimPrefix(parsedURL.Host, "www.")

	targetPrice, err := decimal.NewFromString(targetPriceStr)
	if err != nil {
		return err
	}

	if err = s.repo.CreateUser(ctx, userID); err != nil {
		return err
	}

	product := models.Product{
		URL:          rawURL,
		Domain:       domain,
		CurrentPrice: decimal.NewFromInt(0), // TODO сделать сюда запрос в апи чтобы узнать текущую цену
	}

	productID, err := s.repo.UpsertProduct(ctx, product)

	sub := models.Subscription{
		UserID:      userID,
		ProductID:   productID,
		TargetPrice: targetPrice,
	}

	if err = s.repo.AddSubscription(ctx, sub); err != nil {
		return err
	}
	return nil
}

func (s trackerService) GetUserList(ctx context.Context, userID int64) ([]models.UserSubscription, error) {
	subs, err := s.repo.GetUserSubscriptions(ctx, userID)
	if err != nil {
		return nil, err
	}
	return subs, nil
}
