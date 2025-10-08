package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Address struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      string             `bson:"userId,omitempty" json:"userId"`
	CompanyID   *string            `bson:"companyId,omitempty" json:"companyId"`
	Street      string             `bson:"street" json:"street"`
	PostalCode  string             `bson:"postalCode" json:"postalCode"`
	City        string             `bson:"city" json:"city"`
	Country     string             `bson:"country" json:"country"`
	Type        string             `bson:"type" json:"type"` // ✅ <-- Champ ajouté ici
	IsDefault   bool               `bson:"isDefault" json:"isDefault"`
}