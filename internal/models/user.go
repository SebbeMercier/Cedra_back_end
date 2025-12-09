package models

type User struct {
	ID             string         `json:"user_id"`
	Name           string         `json:"name,omitempty"`
	Email          string         `json:"email"`
	Password       string         `json:"-"`
	Role           string         `json:"role,omitempty"`
	Provider       string         `json:"provider,omitempty"`  // Provider principal (pour compatibilité)
	ProviderID     string         `json:"-"`                   // Provider ID principal
	Providers      []UserProvider `json:"providers,omitempty"` // ✅ Liste de tous les providers
	CompanyID      *string        `json:"companyId,omitempty"`
	CompanyName    string         `json:"companyName,omitempty"`
	IsCompanyAdmin *bool          `json:"isCompanyAdmin,omitempty"`
}

// UserProvider représente une méthode de connexion
type UserProvider struct {
	Provider   string `json:"provider"`    // "local", "google", "facebook"
	ProviderID string `json:"provider_id"` // ID externe ou vide pour local
	LinkedAt   string `json:"linked_at"`   // Date de liaison
}
