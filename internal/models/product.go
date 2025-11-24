package models

import (
	"time"

	"github.com/gocql/gocql"
)

type Product struct {
	ID          gocql.UUID `json:"id" db:"product_id"`
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description" db:"description"`
	Price       float64    `json:"price" db:"price"`
	Stock       int        `json:"stock" db:"stock"`
	CategoryID  gocql.UUID `json:"category_id" db:"category_id"`
	ImageURLs   []string   `json:"image_urls" db:"image_urls"`
	Tags        []string   `json:"tags" db:"tags"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"` // ✅ Pas de pointeur
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"` // ✅ Pas de pointeur
}
