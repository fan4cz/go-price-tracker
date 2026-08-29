package scraper

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/shopspring/decimal"
)

type Scraper interface {
	FetchPrice(ctx context.Context, productURL, domain string) (decimal.Decimal, error)
}

type webScraper struct {
	client    *http.Client
	selectors map[string]string
}

func NewScraper() Scraper {
	return &webScraper{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		selectors: map[string]string{
			"scrapeme.live": ".price",
			"dns-shop.ru":   ".product-buy__price",
			"citilink.ru":   ".ProductHeader__price-default_current-price",
		},
	}
}

func (s *webScraper) FetchPrice(ctx context.Context, productURL, domain string) (decimal.Decimal, error) {
	selector, exists := s.selectors[domain]
	if !exists {
		return decimal.Zero, fmt.Errorf("домен %s пока не поддерживается парсером", domain)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, productURL, nil)
	if err != nil {
		return decimal.Zero, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7")

	res, err := s.client.Do(req)
	if err != nil {
		return decimal.Zero, fmt.Errorf("ошибка сети при запросе к сайту: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return decimal.Zero, fmt.Errorf("сайт вернул статус: %d", res.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return decimal.Zero, fmt.Errorf("ошибка чтения HTML: %w", err)
	}

	priceText := doc.Find(selector).First().Text()
	if priceText == "" {
		return decimal.Zero, fmt.Errorf("цена не найдена по селектору '%s'", selector)
	}

	return cleanPrice(priceText)
}

func cleanPrice(rawPrice string) (decimal.Decimal, error) {
	re := regexp.MustCompile(`[^\d.,]`)
	cleanStr := re.ReplaceAllString(rawPrice, "")

	cleanStr = strings.ReplaceAll(cleanStr, ",", ".")

	if cleanStr == "" {
		return decimal.Zero, fmt.Errorf("после очистки строки '%s' не осталось цифр", rawPrice)
	}

	price, err := decimal.NewFromString(cleanStr)
	if err != nil {
		return decimal.Zero, fmt.Errorf("ошибка конвертации '%s' в число: %w", cleanStr, err)
	}

	return price, nil
}
