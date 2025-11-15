package models

import (
	"github.com/gocql/gocql"
)

type Address struct {
	ID         gocql.UUID `json:"id"`
	UserID     string     `json:"userId"`
	CompanyID  *string    `json:"companyId,omitempty"`
	Street     string     `json:"street"`
	PostalCode string     `json:"postalCode"`
	City       string     `json:"city"`
	Country    string     `json:"country"`
	Type       string     `json:"type"`
	IsDefault  bool       `json:"isDefault"`
}