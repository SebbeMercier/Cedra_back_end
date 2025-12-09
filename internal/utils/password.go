package utils

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Paramètres Argon2id optimisés pour la performance
// Ces paramètres offrent un bon équilibre sécurité/performance
const (
	// Temps de calcul : ~15-20ms (optimisé pour login rapide)
	Argon2Time    = 1         // Nombre d'itérations
	Argon2Memory  = 32 * 1024 // 32 MB de mémoire (réduit de 64 MB)
	Argon2Threads = 4         // Nombre de threads parallèles
	Argon2KeyLen  = 32        // Longueur de la clé (256 bits)
	Argon2SaltLen = 16        // Longueur du salt
)

// HashPassword hash un mot de passe avec Argon2id
func HashPassword(password string) (string, error) {
	// Générer un salt aléatoire
	salt := make([]byte, Argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// Hasher le mot de passe
	hash := argon2.IDKey([]byte(password), salt, Argon2Time, Argon2Memory, Argon2Threads, Argon2KeyLen)

	// Encoder en base64 pour le stockage
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Format: $argon2id$v=19$m=65536,t=1,p=4$salt$hash
	encodedHash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, Argon2Memory, Argon2Time, Argon2Threads, b64Salt, b64Hash)

	return encodedHash, nil
}

// VerifyPassword vérifie si un mot de passe correspond au hash
func VerifyPassword(password, encodedHash string) (bool, error) {
	// Parser le hash
	parts := strings.Split(encodedHash, "$")

	// Support bcrypt pour la rétrocompatibilité
	if strings.HasPrefix(encodedHash, "$2a$") || strings.HasPrefix(encodedHash, "$2b$") {
		// C'est un hash bcrypt, utiliser bcrypt
		return verifyBcrypt(password, encodedHash)
	}

	if len(parts) != 6 {
		return false, errors.New("hash invalide")
	}

	var version int
	var memory, time uint32
	var threads uint8

	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return false, err
	}

	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return false, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}

	// Calculer le hash avec les mêmes paramètres
	otherHash := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(hash)))

	// Comparaison en temps constant
	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		return true, nil
	}

	return false, nil
}

// verifyBcrypt vérifie un hash bcrypt (rétrocompatibilité)
func verifyBcrypt(_, hash string) (bool, error) {
	// Import bcrypt seulement si nécessaire
	// Pour l'instant, retourner false pour forcer la migration
	return false, errors.New("bcrypt non supporté, veuillez réinitialiser votre mot de passe")
}

// IsArgon2Hash vérifie si un hash est au format Argon2
func IsArgon2Hash(hash string) bool {
	return strings.HasPrefix(hash, "$argon2id$")
}

// IsBcryptHash vérifie si un hash est au format bcrypt
func IsBcryptHash(hash string) bool {
	return strings.HasPrefix(hash, "$2a$") || strings.HasPrefix(hash, "$2b$")
}
