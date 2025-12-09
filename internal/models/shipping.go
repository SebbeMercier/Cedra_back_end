package models

type ShippingOption struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	Price         float64 `json:"price"`
	EstimatedDays int     `json:"estimated_days"`
}

type ShippingCalculation struct {
	Options       []ShippingOption `json:"options"`
	FreeThreshold float64          `json:"free_threshold"`
	CartTotal     float64          `json:"cart_total"`
	IsFree        bool             `json:"is_free"`
}
