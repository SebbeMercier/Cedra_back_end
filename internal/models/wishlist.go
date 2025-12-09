package models

import (
	"time"

	"github.com/gocql/gocql"
)

type WishlistItem struct {
	UserID    string     `json:"user_id" db:"user_id"`
	ProductID gocql.UUID `json:"product_id" db:"product_id"`
	AddedAt   time.Time  `json:"added_at" db:"added_at"`
}

type Wishlist struct {
	UserID string    `json:"user_id"`
	Items  []Product `json:"items"`
}
