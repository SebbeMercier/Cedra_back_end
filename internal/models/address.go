package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Address struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID     string             `json:"userId" bson:"userId"`
	Street     string             `json:"street" bson:"street"`
	City       string             `json:"city" bson:"city"`
	PostalCode string             `json:"postalCode" bson:"postalCode"`
	Country    string             `json:"country" bson:"country"`
	IsDefault  bool               `json:"isDefault" bson:"isDefault"`
}