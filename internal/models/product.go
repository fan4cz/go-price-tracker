package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Product struct {
	ID            uint            `db:"id"`
	Url           string          `db:"url"`
	Domain        string          `db:"domain"`
	CurrentPrice  decimal.Decimal `db:"current_price"`
	LastCheckedAt *time.Time      `db:"last_checked_at"`
}
