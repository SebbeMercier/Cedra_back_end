package models

import (
    "go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
    ID             primitive.ObjectID `bson:"_id,omitempty"`
    Name           string             `bson:"name,omitempty"`
    Email          string             `bson:"email"`
    Password       string             `bson:"password,omitempty"`
    Role           string             `bson:"role"`
    IsCompanyAdmin bool               `bson:"isCompanyAdmin"`
    Provider       string             `bson:"provider"`
    ProviderID     string             `bson:"provider_id"`
    CreatedAt      primitive.DateTime `bson:"createdAt"`
}
