package models

import "github.com/shopspring/decimal"

type UserSubscription struct {
	Product
	TargetPrice decimal.Decimal `db:"target_price"`
}
