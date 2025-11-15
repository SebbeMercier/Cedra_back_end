package models

type User struct {
	ID             string  `json:"user_id"`
	Name           string  `json:"name,omitempty"`
	Email          string  `json:"email"`
	Password       string  `json:"-"`
	Role           string  `json:"role,omitempty"`
	Provider       string  `json:"provider,omitempty"`
	ProviderID     string  `json:"-"`
	CompanyID      *string `json:"companyId,omitempty"`
	CompanyName    string  `json:"companyName,omitempty"`
	IsCompanyAdmin *bool   `json:"isCompanyAdmin,omitempty"`
}
