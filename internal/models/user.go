package models

type User struct {
	ID             string  `bson:"_id,omitempty" json:"user_id"`
	Name           string  `json:"name,omitempty"`
	Email          string  `json:"email"`
	Password       string  `bson:"password,omitempty" json:"-"`
	Role           string  `json:"role,omitempty"`
	Provider       string  `json:"provider,omitempty"`
	ProviderID     string  `bson:"provider_id,omitempty" json:"-"`
	CompanyID      *string `bson:"companyId,omitempty" json:"companyId,omitempty"`
	CompanyName    string  `json:"companyName,omitempty"`
	IsCompanyAdmin *bool   `json:"isCompanyAdmin,omitempty"`
}
