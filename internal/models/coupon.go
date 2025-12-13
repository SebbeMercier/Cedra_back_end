package models

import (
	"time"

	"github.com/gocql/gocql"
)

type Coupon struct {
	ID              gocql.UUID `json:"id"`
	Code            string     `json:"code"`
	Type            string     `json:"type"` // "percentage", "fixed", "free_shipping"
	Value           float64    `json:"value"`
	MinAmount       float64    `json:"min_amount"`
	MaxAmount       *float64   `json:"max_amount,omitempty"` // Montant max de réduction
	MaxUses         int        `json:"max_uses"`
	UsedCount       int        `json:"used_count"`
	MaxUsesPerUser  int        `json:"max_uses_per_user"`
	ApplicableToAll bool       `json:"applicable_to_all"`
	ProductIDs      []string   `json:"product_ids,omitempty"`
	CategoryIDs     []string   `json:"category_ids,omitempty"`
	ExpiresAt       time.Time  `json:"expires_at"`
	StartsAt        time.Time  `json:"starts_at"`
	IsActive        bool       `json:"is_active"`
	CreatedBy       string     `json:"created_by"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type CouponUsage struct {
	ID       gocql.UUID `json:"id"`
	CouponID gocql.UUID `json:"coupon_id"`
	UserID   string     `json:"user_id"`
	OrderID  gocql.UUID `json:"order_id"`
	UsedAt   time.Time  `json:"used_at"`
}

type CouponValidation struct {
	IsValid      bool    `json:"is_valid"`
	ErrorMessage string  `json:"error_message,omitempty"`
	Discount     float64 `json:"discount"`
	Type         string  `json:"type"`
	Code         string  `json:"code"`
}
