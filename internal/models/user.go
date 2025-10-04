package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name           string             `bson:"name" json:"name"`
	Email          string             `bson:"email" json:"email"`
	Password       string             `bson:"password" json:"-"`
	Role           string             `bson:"role" json:"role"`
	CompanyID      *string            `bson:"companyId,omitempty" json:"companyId,omitempty"`
	IsCompanyAdmin bool               `bson:"isCompanyAdmin" json:"isCompanyAdmin"`
	Provider       string             `bson:"provider" json:"provider"`
	ProviderID     string             `bson:"provider_id,omitempty" json:"provider_id,omitempty"`
}