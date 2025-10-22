package models

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Order struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID          string             `bson:"user_id" json:"user_id"`
	PaymentIntentID string             `bson:"payment_intent_id" json:"payment_intent_id"`
	Items           []OrderItem        `bson:"items" json:"items"`
	TotalPrice      float64            `bson:"total_price" json:"total_price"`
	Status          string             `bson:"status" json:"status"` // "pending", "paid", "shipped", "delivered"
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time          `bson:"updated_at,omitempty" json:"updated_at,omitempty"`
}

type OrderItem struct {
    ProductID string  `bson:"product_id" json:"productId"`
    Quantity  int     `bson:"quantity" json:"quantity"`
    Price     float64 `bson:"price" json:"price"`
    Name      string  `bson:"name" json:"name"`
}