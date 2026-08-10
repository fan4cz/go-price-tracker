package models

import "github.com/shopspring/decimal"

type Substriction struct {
	UserID uint `db:"user_id"`
	Product uint `db:"product_id"`
	TargetPrice decimal.Decimal `db:"target_price"`
}
