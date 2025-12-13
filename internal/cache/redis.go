package cache

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	RedisClient *redis.Client
	ctx         = context.Background()
)

// InitRedis initialise la connexion Redis
func InitRedis() error {
	redisHost := os.Getenv("REDIS_HOST")
	redisPassword := os.Getenv("REDIS_PASSWORD")

	if redisHost == "" {
		return fmt.Errorf("REDIS_HOST non configuré")
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr:         redisHost,
		Password:     redisPassword,
		DB:           0, // Base de données par défaut
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	// Test de connexion
	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("impossible de se connecter à Redis: %v", err)
	}

	log.Println("✅ Redis connecté avec succès")
	return nil
}

// CloseRedis ferme la connexion Redis
func CloseRedis() error {
	if RedisClient != nil {
		return RedisClient.Close()
	}
	return nil
}

// --- Refresh Tokens ---

// StoreRefreshToken stocke un refresh token pour un utilisateur
func StoreRefreshToken(userID, refreshToken string, duration time.Duration) error {
	key := fmt.Sprintf("refresh:%s", userID)
	return RedisClient.Set(ctx, key, refreshToken, duration).Err()
}

// GetRefreshToken récupère le refresh token d'un utilisateur
func GetRefreshToken(userID string) (string, error) {
	key := fmt.Sprintf("refresh:%s", userID)
	return RedisClient.Get(ctx, key).Result()
}

// DeleteRefreshToken supprime le refresh token (logout)
func DeleteRefreshToken(userID string) error {
	key := fmt.Sprintf("refresh:%s", userID)
	return RedisClient.Del(ctx, key).Err()
}

// DeleteAllRefreshTokens supprime tous les refresh tokens d'un user (logout all devices)
func DeleteAllRefreshTokens(userID string) error {
	pattern := fmt.Sprintf("refresh:%s:*", userID)
	keys, err := RedisClient.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	
	if len(keys) > 0 {
		return RedisClient.Del(ctx, keys...).Err()
	}
	
	// Supprimer aussi le refresh token principal
	return DeleteRefreshToken(userID)
}

// --- Blacklist JWT (révocation avant expiration) ---

// BlacklistToken ajoute un token JWT à la blacklist
func BlacklistToken(tokenID string, duration time.Duration) error {
	key := fmt.Sprintf("blacklist:%s", tokenID)
	return RedisClient.Set(ctx, key, "revoked", duration).Err()
}

// IsTokenBlacklisted vérifie si un token est blacklisté
func IsTokenBlacklisted(tokenID string) bool {
	key := fmt.Sprintf("blacklist:%s", tokenID)
	exists, err := RedisClient.Exists(ctx, key).Result()
	if err != nil {
		log.Printf("⚠️ Erreur vérification blacklist: %v", err)
		return false
	}
	return exists > 0
}

// --- Ban utilisateurs ---

// BanUser bannit un utilisateur (révocation permanente)
func BanUser(userID string) error {
	key := fmt.Sprintf("banned:%s", userID)
	// Pas d'expiration = permanent
	return RedisClient.Set(ctx, key, "true", 0).Err()
}

// UnbanUser débannit un utilisateur
func UnbanUser(userID string) error {
	key := fmt.Sprintf("banned:%s", userID)
	return RedisClient.Del(ctx, key).Err()
}

// IsUserBanned vérifie si un utilisateur est banni
func IsUserBanned(userID string) bool {
	key := fmt.Sprintf("banned:%s", userID)
	exists, err := RedisClient.Exists(ctx, key).Result()
	if err != nil {
		log.Printf("⚠️ Erreur vérification ban: %v", err)
		return false
	}
	return exists > 0
}

// --- Sessions multi-device ---

// StoreDeviceSession stocke une session pour un device spécifique
func StoreDeviceSession(userID, deviceID, refreshToken string, duration time.Duration) error {
	key := fmt.Sprintf("refresh:%s:%s", userID, deviceID)
	return RedisClient.Set(ctx, key, refreshToken, duration).Err()
}

// GetDeviceSession récupère la session d'un device
func GetDeviceSession(userID, deviceID string) (string, error) {
	key := fmt.Sprintf("refresh:%s:%s", userID, deviceID)
	return RedisClient.Get(ctx, key).Result()
}

// DeleteDeviceSession supprime la session d'un device (logout device)
func DeleteDeviceSession(userID, deviceID string) error {
	key := fmt.Sprintf("refresh:%s:%s", userID, deviceID)
	return RedisClient.Del(ctx, key).Err()
}

// GetAllUserDevices récupère tous les devices d'un utilisateur
func GetAllUserDevices(userID string) ([]string, error) {
	pattern := fmt.Sprintf("refresh:%s:*", userID)
	keys, err := RedisClient.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}
	
	devices := make([]string, 0, len(keys))
	for _, key := range keys {
		// Extraire le deviceID du pattern "refresh:userID:deviceID"
		// On skip le refresh token principal (sans deviceID)
		if len(key) > len(pattern)-1 {
			devices = append(devices, key)
		}
	}
	
	return devices, nil
}

// --- Cache générique ---

// SetCache stocke une valeur dans le cache
func SetCache(key string, value interface{}, duration time.Duration) error {
	return RedisClient.Set(ctx, key, value, duration).Err()
}

// GetCache récupère une valeur du cache
func GetCache(key string) (string, error) {
	return RedisClient.Get(ctx, key).Result()
}

// DeleteCache supprime une clé du cache
func DeleteCache(key string) error {
	return RedisClient.Del(ctx, key).Err()
}

// --- Rate Limiting Global ---

// IncrementRateLimit incrémente le compteur de rate limit
func IncrementRateLimit(key string, window time.Duration) (int64, error) {
	pipe := RedisClient.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	return incr.Val(), nil
}

// GetRateLimit récupère le compteur de rate limit
func GetRateLimit(key string) (int64, error) {
	val, err := RedisClient.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}
