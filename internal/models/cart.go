package models

type Cart struct {
	UserID string     `json:"user_id"`
	Items  []CartItem `json:"items"`
}

type CartItem struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name" bson:"name"` 
	Price     float64 `json:"price"`
	Quantity  int     `json:"quantity"`
	ImageURL  string  `json:"image_url"`
}