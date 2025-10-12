package config

import (
	"log"

	"github.com/joho/godotenv"
)

func Load() {
	// Charge .env
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("⚠️  Aucun fichier .env trouvé — on continue avec les variables d'environnement du système")
	} else {
		log.Println("✅ Fichier .env chargé avec succès")
	}
}
