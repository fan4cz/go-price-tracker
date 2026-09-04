package models

type ScrapeJobEvent struct {
	ProductID int    `json:"product_id"`
	URL       string `json:"url"`
	Domain    string `json:"domain"`
}

type AlertEvent struct {
	UserID      int64  `json:"user_id"`
	URL         string `json:"url"`
	NewPrice    string `json:"new_price"`
	TargetPrice string `json:"target_price"`
}
