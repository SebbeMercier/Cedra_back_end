package models

import (
	"time"

	"github.com/gocql/gocql"
)

type Product struct {
	ID                gocql.UUID `json:"id" db:"product_id"`
	Name              string     `json:"name" db:"name"`
	Description       string     `json:"description" db:"description"`
	Price             float64    `json:"price" db:"price"`
	Stock             int        `json:"stock" db:"stock"`
	LowStockThreshold int        `json:"low_stock_threshold" db:"low_stock_threshold"`
	SKU               string     `json:"sku" db:"sku"`
	Weight            float64    `json:"weight" db:"weight"`
	CategoryID        gocql.UUID `json:"category_id" db:"category_id"`
	ImageURLs         []string   `json:"image_urls" db:"image_urls"`
	Tags              []string   `json:"tags" db:"tags"`
	IsActive          bool       `json:"is_active" db:"is_active"`
	HasVariants       bool       `json:"has_variants" db:"has_variants"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
}
