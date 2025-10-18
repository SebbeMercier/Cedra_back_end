package services

import (
	"context"
	"net/url"
	"strings"
	"time"

	"cedra_back_end/internal/database"
)

// ✅ Version améliorée : supporte la durée et le contexte
func GenerateSignedURL(ctx context.Context, objectPath string, duration time.Duration) (string, error) {
	reqParams := make(url.Values)

	// Nettoie l'URL complète pour ne garder que le chemin relatif à ton bucket
	key := strings.TrimPrefix(objectPath, "http://192.168.1.130:9000/cedra-images/")

	// Génère l'URL signée avec expiration
	presignedURL, err := database.MinioClient.PresignedGetObject(
		ctx,
		"cedra-images",
		key,
		duration,
		reqParams,
	)
	if err != nil {
		return "", err
	}

	return presignedURL.String(), nil
}