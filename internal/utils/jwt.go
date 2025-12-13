package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

// Claims JWT personnalisés
type JWTClaims struct {
	UserID         string `json:"user_id"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	IsCompanyAdmin bool   `json:"isCompanyAdmin"`
	TokenID        string `json:"jti"` // JWT ID pour blacklist
	jwt.RegisteredClaims
}

// GenerateAccessToken génère un access token JWT (courte durée)
func GenerateAccessToken(userID, email, role string, isCompanyAdmin bool) (string, string, error) {
	tokenID := uuid.New().String() // ID unique pour blacklist

	claims := JWTClaims{
		UserID:         userID,
		Email:          email,
		Role:           role,
		IsCompanyAdmin: isCompanyAdmin,
		TokenID:        tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)), // 15 minutes
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "cedra-backend",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", "", err
	}

	return tokenString, tokenID, nil
}

// GenerateRefreshToken génère un refresh token aléatoire (longue durée)
func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// ParseAccessToken parse et valide un access token JWT
func ParseAccessToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("méthode de signature inattendue: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("token invalide")
}

// GetTokenExpirationDuration retourne la durée restante avant expiration
func GetTokenExpirationDuration(claims *JWTClaims) time.Duration {
	return time.Until(claims.ExpiresAt.Time)
}
