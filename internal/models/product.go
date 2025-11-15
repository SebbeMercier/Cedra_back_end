package models

import (
	"time"
	"github.com/gocql/gocql"
)

type Product struct {
	ID          gocql.UUID `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Price       float64    `json:"price"`
	Stock       int        `json:"stock"`
	CategoryID  gocql.UUID `json:"category_id,omitempty"`
	ImageURLs   []string   `json:"image_urls"`
	Tags        []string   `json:"tags,omitempty"`
	CompanyID   gocql.UUID `json:"company_id,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}
