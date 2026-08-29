package models

type ScrapeJobEvent struct {
	ProductID int    `json:"product_id"`
	URL       string `json:"url"`
	Domain    string `json:"domain"`
}
