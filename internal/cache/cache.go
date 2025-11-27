package cache

import (
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
	"context"
	"encoding/json"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

const (
	UserCacheTTL    = 5 * time.Minute
	ProductCacheTTL = 10 * time.Minute
)

// GetUserFromCache récupère un utilisateur depuis Redis ou ScyllaDB
func GetUserFromCache(userID string) (*models.User, error) {
	ctx := context.Background()
	key := "user:" + userID

	// 1. Essayer le cache Redis
	data, err := database.Redis.Get(ctx, key).Result()
	if err == nil {
		var user models.User
		if json.Unmarshal([]byte(data), &user) == nil {
			return &user, nil
		}
	}

	// 2. Récupérer de ScyllaDB
	session, err := database.GetUsersSession()
	if err != nil {
		return nil, err
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	userUUID := gocql.UUID(uid)

	var (
		email, name, role, provider string
		companyID                   *gocql.UUID
		companyName                 string
		isCompanyAdmin              bool
	)

	err = session.Query(`SELECT email, name, role, provider, company_id, company_name, is_company_admin 
		FROM users WHERE user_id = ?`, userUUID).Scan(
		&email, &name, &role, &provider, &companyID, &companyName, &isCompanyAdmin)
	if err != nil {
		return nil, err
	}

	var companyIDStr *string
	if companyID != nil {
		s := companyID.String()
		companyIDStr = &s
	}

	user := &models.User{
		ID:             userID,
		Email:          email,
		Name:           name,
		Role:           role,
		Provider:       provider,
		CompanyID:      companyIDStr,
		CompanyName:    companyName,
		IsCompanyAdmin: &isCompanyAdmin,
	}

	// 3. Mettre en cache
	jsonData, _ := json.Marshal(user)
	database.Redis.Set(ctx, key, jsonData, UserCacheTTL)

	return user, nil
}

// InvalidateUserCache invalide le cache d'un utilisateur
func InvalidateUserCache(userID string) {
	ctx := context.Background()
	database.Redis.Del(ctx, "user:"+userID)
}

// GetProductNamesFromCache récupère plusieurs noms de produits
func GetProductNamesFromCache(productIDs []string) map[string]string {
	ctx := context.Background()
	result := make(map[string]string)
	missingIDs := []string{}

	// 1. Essayer de récupérer depuis Redis
	for _, productID := range productIDs {
		key := "product_name:" + productID
		name, err := database.Redis.Get(ctx, key).Result()
		if err == nil {
			result[productID] = name
		} else {
			missingIDs = append(missingIDs, productID)
		}
	}

	// 2. Récupérer les produits manquants depuis ScyllaDB
	if len(missingIDs) > 0 {
		session, err := database.GetProductsSession()
		if err == nil {
			for _, productID := range missingIDs {
				pid, err := uuid.Parse(productID)
				if err == nil {
					var name string
					err = session.Query("SELECT name FROM products WHERE product_id = ?", gocql.UUID(pid)).Scan(&name)
					if err == nil {
						result[productID] = name
						// Mettre en cache
						database.Redis.Set(ctx, "product_name:"+productID, name, ProductCacheTTL)
					}
				}
			}
		}
	}

	return result
}

// InvalidateProductCache invalide le cache d'un produit
func InvalidateProductCache(productID string) {
	ctx := context.Background()
	database.Redis.Del(ctx, "product:"+productID, "product_name:"+productID)
}
