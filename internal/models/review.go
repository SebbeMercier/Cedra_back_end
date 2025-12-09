package models

import (
	"time"

	"github.com/gocql/gocql"
)

type Review struct {
	ID        gocql.UUID `json:"id" db:"review_id"`
	ProductID gocql.UUID `json:"product_id" db:"product_id"`
	UserID    string     `json:"user_id" db:"user_id"`
	UserName  string     `json:"user_name" db:"user_name"`
	Rating    int        `json:"rating" db:"rating"` // 1-5
	Comment   string     `json:"comment" db:"comment"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

type ProductRating struct {
	ProductID    gocql.UUID `json:"product_id"`
	AverageRating float64   `json:"average_rating"`
	TotalReviews int        `json:"total_reviews"`
}
