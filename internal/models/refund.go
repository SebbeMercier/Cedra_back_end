package models

import (
	"time"

	"github.com/gocql/gocql"
)

type Refund struct {
	ID             gocql.UUID `json:"id" db:"refund_id"`
	OrderID        gocql.UUID `json:"order_id" db:"order_id"`
	UserID         string     `json:"user_id" db:"user_id"`
	Reason         string     `json:"reason" db:"reason"`
	Status         string     `json:"status" db:"status"` // pending, approved, rejected, completed
	RefundAmount   float64    `json:"refund_amount" db:"refund_amount"`
	StripeRefundID string     `json:"stripe_refund_id,omitempty" db:"stripe_refund_id"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty" db:"updated_at"`
}
