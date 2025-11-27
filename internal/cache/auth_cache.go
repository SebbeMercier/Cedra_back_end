package cache

import (
	"cedra_back_end/internal/database"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

const (
	AuthCacheTTL = 15 * time.Minute // Cache les vérifications de mot de passe pendant 15 min
)

// GetPasswordHashFromCache vérifie si le hash du mot de passe est en cache
// Cela évite de refaire bcrypt.CompareHashAndPassword à chaque login
func GetPasswordHashFromCache(email, password string) (bool, error) {
	ctx := context.Background()

	// Créer une clé unique basée sur email + hash du password
	passwordHash := sha256.Sum256([]byte(password))
	cacheKey := "auth:" + email + ":" + hex.EncodeToString(passwordHash[:])

	// Vérifier si cette combinaison est en cache
	result, err := database.Redis.Get(ctx, cacheKey).Result()
	if err == nil && result == "valid" {
		return true, nil
	}

	return false, err
}

// SetPasswordHashInCache met en cache une vérification de mot de passe réussie
func SetPasswordHashInCache(email, password string) {
	ctx := context.Background()

	passwordHash := sha256.Sum256([]byte(password))
	cacheKey := "auth:" + email + ":" + hex.EncodeToString(passwordHash[:])

	// Mettre en cache pendant 15 minutes
	database.Redis.Set(ctx, cacheKey, "valid", AuthCacheTTL)
}

// InvalidateAuthCache invalide le cache d'authentification pour un email
func InvalidateAuthCache(email string) {
	ctx := context.Background()

	// Supprimer toutes les clés auth:email:*
	pattern := "auth:" + email + ":*"
	iter := database.Redis.Scan(ctx, 0, pattern, 100).Iterator()

	for iter.Next(ctx) {
		database.Redis.Del(ctx, iter.Val())
	}
}
