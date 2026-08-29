package service

import (
	"context"
	"fmt"
	"go-price-tracker/internal/models"
	"go-price-tracker/internal/scraper"
	"go-price-tracker/internal/storage"
	"net/url"
	"strings"

	"github.com/shopspring/decimal"
)

type TrackerService interface {
	TrackProduct(ctx context.Context, userID int64, rawURL string, targetPriceStr string) (decimal.Decimal, error)
	GetUserList(ctx context.Context, userID int64) ([]models.UserSubscription, error)
}

type trackerService struct {
	repo    *storage.Repository
	scraper scraper.Scraper
}

func NewTrackerService(repo *storage.Repository, scraper scraper.Scraper) TrackerService {
	return &trackerService{
		repo:    repo,
		scraper: scraper,
	}
}

func (s trackerService) TrackProduct(ctx context.Context, userID int64, rawURL string, targetPriceStr string) (decimal.Decimal, error) {
	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return decimal.Zero, err
	}

	domain := strings.TrimPrefix(parsedURL.Host, "www.")

	targetPrice, err := decimal.NewFromString(targetPriceStr)
	if err != nil {
		return decimal.Zero, err
	}

	curPrice, err := s.scraper.FetchPrice(ctx, rawURL, domain)
	if err != nil {
		return decimal.Zero, fmt.Errorf("не удалось получить цену с сайта %s: %w", domain, err)
	}

	if err = s.repo.CreateUser(ctx, userID); err != nil {
		return decimal.Zero, err
	}

	product := models.Product{
		URL:          rawURL,
		Domain:       domain,
		CurrentPrice: curPrice,
	}

	productID, err := s.repo.UpsertProduct(ctx, product)

	sub := models.Subscription{
		UserID:      userID,
		ProductID:   productID,
		TargetPrice: targetPrice,
	}

	if err = s.repo.AddSubscription(ctx, sub); err != nil {
		return decimal.Zero, err
	}
	return curPrice, nil
}

func (s trackerService) GetUserList(ctx context.Context, userID int64) ([]models.UserSubscription, error) {
	subs, err := s.repo.GetUserSubscriptions(ctx, userID)
	if err != nil {
		return nil, err
	}
	return subs, nil
}
