package models

import (
	"github.com/gocql/gocql"
	"time"
)

type Category struct {
	ID          gocql.UUID `json:"id,omitempty"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Description string     `json:"description,omitempty"`
	ImageURL    string     `json:"image_url,omitempty"`
	ParentCategoryID *gocql.UUID `json:"parent_category_id,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
}
