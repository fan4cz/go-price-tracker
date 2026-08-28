package models

import "github.com/shopspring/decimal"

type Subscription struct {
	UserID      int64           `db:"user_id"`
	ProductID   int             `db:"product_id"`
	TargetPrice decimal.Decimal `db:"target_price"`
}
