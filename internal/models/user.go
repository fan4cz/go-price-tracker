package models

import "time"

type User struct {
	ID        uint64     `db:"tg_bot_id"`
	CreatedAt *time.Time `db:"created_at"`
}
