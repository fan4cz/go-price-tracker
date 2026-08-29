package models

import "time"

type User struct {
	ID        int64      `db:"tg_bot_id"`
	CreatedAt *time.Time `db:"created_at"`
}
