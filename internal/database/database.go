package database

import (
	"log"
	"os"
	"time"

	"cedra_back_end/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() {
	dsn := os.Getenv("DB_URL")
	if dsn == "" {
		log.Fatal("❌ DB_URL manquant dans .env")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal("❌ Impossible de se connecter à PostgreSQL:", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("❌ Erreur lors de la récupération du pool DB:", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	DB = db

	log.Println("✅ Connecté à PostgreSQL")

	// ✅ bien utiliser le modèle importé
	if err := DB.AutoMigrate(&models.User{}); err != nil {
		log.Fatalf("❌ Erreur migration: %v", err)
	}
}