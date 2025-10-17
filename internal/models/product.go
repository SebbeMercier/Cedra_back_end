package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Product struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description" json:"description"`
	Price       float64            `bson:"price" json:"price"`
	Stock       int                `bson:"stock" json:"stock"`
	CategoryID  primitive.ObjectID `bson:"category_id,omitempty" json:"category_id,omitempty"`
	ImageURLs []string `bson:"image_urls" json:"image_urls"`
	Tags        []string           `bson:"tags,omitempty" json:"tags,omitempty"`
}
