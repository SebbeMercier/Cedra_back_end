package models

import (
	"time"
	"github.com/gocql/gocql"
)

type Order struct {
	ID              gocql.UUID `json:"id"`
	UserID          string     `json:"user_id"`
	PaymentIntentID string     `json:"payment_intent_id"`
	Items           []OrderItem `json:"items"`
	TotalPrice      float64    `json:"total_price"`
	Status          string     `json:"status"` // "pending", "paid", "shipped", "delivered"
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
}

type OrderItem struct {
	ProductID   string  `json:"productId"`
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
	Name        string  `json:"name"`
}