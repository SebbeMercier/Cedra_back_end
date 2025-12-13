package models

import (
	"time"

	"github.com/gocql/gocql"
)

type StockMovement struct {
	ID        gocql.UUID  `json:"id"`
	ProductID gocql.UUID  `json:"product_id"`
	Type      string      `json:"type"` // "sale", "restock", "return", "adjustment", "reserved"
	Quantity  int         `json:"quantity"`
	PrevStock int         `json:"prev_stock"`
	NewStock  int         `json:"new_stock"`
	Reason    string      `json:"reason"`
	OrderID   *gocql.UUID `json:"order_id,omitempty"`
	UserID    string      `json:"user_id"`
	CreatedAt time.Time   `json:"created_at"`
}

type ProductVariant struct {
	ID         gocql.UUID        `json:"id"`
	ProductID  gocql.UUID        `json:"product_id"`
	SKU        string            `json:"sku"`
	Price      float64           `json:"price"`
	Stock      int               `json:"stock"`
	Attributes map[string]string `json:"attributes"` // {"size": "L", "color": "red"}
	IsActive   bool              `json:"is_active"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

type StockAlert struct {
	ID             gocql.UUID `json:"id"`
	ProductID      gocql.UUID `json:"product_id"`
	ProductName    string     `json:"product_name"`
	CurrentStock   int        `json:"current_stock"`
	ThresholdStock int        `json:"threshold_stock"`
	AlertType      string     `json:"alert_type"` // "low_stock", "out_of_stock"
	IsResolved     bool       `json:"is_resolved"`
	CreatedAt      time.Time  `json:"created_at"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
}

type InventoryStats struct {
	TotalProducts      int            `json:"total_products"`
	LowStockProducts   int            `json:"low_stock_products"`
	OutOfStockProducts int            `json:"out_of_stock_products"`
	TotalValue         float64        `json:"total_value"`
	TopSellingProducts []ProductSales `json:"top_selling_products"`
}

type ProductSales struct {
	ProductID   gocql.UUID `json:"product_id"`
	ProductName string     `json:"product_name"`
	TotalSold   int        `json:"total_sold"`
	Revenue     float64    `json:"revenue"`
}
