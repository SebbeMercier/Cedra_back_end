package models

import (
	"time"

	"github.com/gocql/gocql"
)

// Role représente un rôle avec ses permissions
type Role struct {
	ID          gocql.UUID `json:"id"`
	Name        string     `json:"name"`
	DisplayName string     `json:"display_name"`
	Description string     `json:"description"`
	Permissions []string   `json:"permissions"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// UserRole représente l'attribution d'un rôle à un utilisateur
type UserRole struct {
	ID        gocql.UUID `json:"id"`
	UserID    string     `json:"user_id"`
	RoleID    gocql.UUID `json:"role_id"`
	RoleName  string     `json:"role_name"`
	GrantedBy string     `json:"granted_by"`
	GrantedAt time.Time  `json:"granted_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	IsActive  bool       `json:"is_active"`
}

// Permission représente une permission spécifique
type Permission struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// AuditLog représente un log d'audit pour tracer les actions
type AuditLog struct {
	ID         gocql.UUID `json:"id"`
	UserID     string     `json:"user_id"`
	UserEmail  string     `json:"user_email"`
	Action     string     `json:"action"`
	Resource   string     `json:"resource"`
	ResourceID string     `json:"resource_id,omitempty"`
	OldValue   string     `json:"old_value,omitempty"`
	NewValue   string     `json:"new_value,omitempty"`
	IPAddress  string     `json:"ip_address"`
	UserAgent  string     `json:"user_agent"`
	Success    bool       `json:"success"`
	ErrorMsg   string     `json:"error_msg,omitempty"`
	Timestamp  time.Time  `json:"timestamp"`
	SessionID  string     `json:"session_id,omitempty"`
}

// Permissions prédéfinies
var (
	// Gestion des produits
	PERM_PRODUCTS_VIEW   = "products.view"
	PERM_PRODUCTS_CREATE = "products.create"
	PERM_PRODUCTS_EDIT   = "products.edit"
	PERM_PRODUCTS_DELETE = "products.delete"
	PERM_PRODUCTS_PRICE  = "products.price"

	// Gestion des commandes
	PERM_ORDERS_VIEW   = "orders.view"
	PERM_ORDERS_EDIT   = "orders.edit"
	PERM_ORDERS_CANCEL = "orders.cancel"
	PERM_ORDERS_REFUND = "orders.refund"

	// Gestion des utilisateurs
	PERM_USERS_VIEW   = "users.view"
	PERM_USERS_CREATE = "users.create"
	PERM_USERS_EDIT   = "users.edit"
	PERM_USERS_DELETE = "users.delete"
	PERM_USERS_BAN    = "users.ban"

	// Gestion financière
	PERM_FINANCE_VIEW     = "finance.view"
	PERM_FINANCE_INVOICES = "finance.invoices"
	PERM_FINANCE_REPORTS  = "finance.reports"
	PERM_FINANCE_REFUNDS  = "finance.refunds"

	// Gestion des coupons
	PERM_COUPONS_VIEW   = "coupons.view"
	PERM_COUPONS_CREATE = "coupons.create"
	PERM_COUPONS_EDIT   = "coupons.edit"
	PERM_COUPONS_DELETE = "coupons.delete"

	// Gestion de l'inventaire
	PERM_INVENTORY_VIEW   = "inventory.view"
	PERM_INVENTORY_EDIT   = "inventory.edit"
	PERM_INVENTORY_ALERTS = "inventory.alerts"

	// Analytics et rapports
	PERM_ANALYTICS_VIEW     = "analytics.view"
	PERM_ANALYTICS_ADVANCED = "analytics.advanced"
	PERM_REPORTS_VIEW       = "reports.view"
	PERM_REPORTS_EXPORT     = "reports.export"

	// Administration système
	PERM_ADMIN_ROLES       = "admin.roles"
	PERM_ADMIN_PERMISSIONS = "admin.permissions"
	PERM_ADMIN_LOGS        = "admin.logs"
	PERM_ADMIN_SETTINGS    = "admin.settings"
)

// Rôles prédéfinis
var DefaultRoles = []Role{
	{
		Name:        "super_admin",
		DisplayName: "Super Administrateur",
		Description: "Accès complet à toutes les fonctionnalités",
		Permissions: []string{
			PERM_PRODUCTS_VIEW, PERM_PRODUCTS_CREATE, PERM_PRODUCTS_EDIT, PERM_PRODUCTS_DELETE, PERM_PRODUCTS_PRICE,
			PERM_ORDERS_VIEW, PERM_ORDERS_EDIT, PERM_ORDERS_CANCEL, PERM_ORDERS_REFUND,
			PERM_USERS_VIEW, PERM_USERS_CREATE, PERM_USERS_EDIT, PERM_USERS_DELETE, PERM_USERS_BAN,
			PERM_FINANCE_VIEW, PERM_FINANCE_INVOICES, PERM_FINANCE_REPORTS, PERM_FINANCE_REFUNDS,
			PERM_COUPONS_VIEW, PERM_COUPONS_CREATE, PERM_COUPONS_EDIT, PERM_COUPONS_DELETE,
			PERM_INVENTORY_VIEW, PERM_INVENTORY_EDIT, PERM_INVENTORY_ALERTS,
			PERM_ANALYTICS_VIEW, PERM_ANALYTICS_ADVANCED, PERM_REPORTS_VIEW, PERM_REPORTS_EXPORT,
			PERM_ADMIN_ROLES, PERM_ADMIN_PERMISSIONS, PERM_ADMIN_LOGS, PERM_ADMIN_SETTINGS,
		},
		IsActive: true,
	},
	{
		Name:        "finance_manager",
		DisplayName: "Responsable Financier",
		Description: "Gestion des finances, factures et remboursements",
		Permissions: []string{
			PERM_ORDERS_VIEW, PERM_ORDERS_REFUND,
			PERM_FINANCE_VIEW, PERM_FINANCE_INVOICES, PERM_FINANCE_REPORTS, PERM_FINANCE_REFUNDS,
			PERM_ANALYTICS_VIEW, PERM_REPORTS_VIEW, PERM_REPORTS_EXPORT,
		},
		IsActive: true,
	},
	{
		Name:        "inventory_manager",
		DisplayName: "Responsable Stock",
		Description: "Gestion de l'inventaire et des produits",
		Permissions: []string{
			PERM_PRODUCTS_VIEW, PERM_PRODUCTS_CREATE, PERM_PRODUCTS_EDIT,
			PERM_INVENTORY_VIEW, PERM_INVENTORY_EDIT, PERM_INVENTORY_ALERTS,
			PERM_ANALYTICS_VIEW,
		},
		IsActive: true,
	},
	{
		Name:        "marketing_manager",
		DisplayName: "Responsable Marketing",
		Description: "Gestion des promotions et coupons",
		Permissions: []string{
			PERM_PRODUCTS_VIEW,
			PERM_COUPONS_VIEW, PERM_COUPONS_CREATE, PERM_COUPONS_EDIT, PERM_COUPONS_DELETE,
			PERM_ANALYTICS_VIEW, PERM_REPORTS_VIEW,
		},
		IsActive: true,
	},
	{
		Name:        "customer_service",
		DisplayName: "Service Client",
		Description: "Gestion des commandes et support client",
		Permissions: []string{
			PERM_PRODUCTS_VIEW,
			PERM_ORDERS_VIEW, PERM_ORDERS_EDIT, PERM_ORDERS_CANCEL,
			PERM_USERS_VIEW,
			PERM_FINANCE_REFUNDS,
		},
		IsActive: true,
	},
	{
		Name:        "analyst",
		DisplayName: "Analyste",
		Description: "Consultation des données et rapports",
		Permissions: []string{
			PERM_PRODUCTS_VIEW,
			PERM_ORDERS_VIEW,
			PERM_USERS_VIEW,
			PERM_ANALYTICS_VIEW, PERM_ANALYTICS_ADVANCED, PERM_REPORTS_VIEW, PERM_REPORTS_EXPORT,
		},
		IsActive: true,
	},
}
