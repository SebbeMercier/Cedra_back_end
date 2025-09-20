package config

import (
	"log"

	"github.com/joho/godotenv"
)

func Load() {
	err := godotenv.Load(".env") // 👈 précise explicitement le fichier
	if err != nil {
		log.Fatalf("❌ Impossible de charger .env: %v", err)
	}
}