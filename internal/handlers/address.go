package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

//
// --- HANDLERS ADRESSES ---
//

// 🟢 GET /api/addresses/mine
func ListMyAddresses(c *gin.Context) {
	userID := c.GetString("user_id")
	companyID := c.GetString("company_id") // 🔹 si présent dans le JWT/middleware
	log.Printf("🔍 DEBUG /addresses/mine → user_id=%v, company_id=%v", userID, companyID)

	if userID == "" {
		log.Println("❌ Aucun user_id trouvé dans le contexte (JWT invalide ?)")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "non authentifié"})
		return
	}

	col := database.MongoAddressesDB.Collection("addresses")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 🔍 Recherche toutes les adresses personnelles OU liées à la société
	filter := bson.M{
		"$or": []bson.M{
			{"userId": userID},
			{"companyId": companyID},
		},
		"type": bson.M{"$ne": "billing"}, // exclut les adresses de facturation
	}

	cursor, err := col.Find(ctx, filter)
	if err != nil {
		log.Println("❌ Erreur Mongo Find:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erreur de lecture"})
		return
	}
	defer cursor.Close(ctx)

	var results []models.Address
	if err := cursor.All(ctx, &results); err != nil {
		log.Println("❌ Erreur décodage:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erreur décodage adresses"})
		return
	}

	log.Printf("✅ %d adresses trouvées pour user %s", len(results), userID)
	c.JSON(http.StatusOK, results)
}

// 🟢 POST /api/addresses
func CreateAddress(c *gin.Context) {
	userID := c.GetString("user_id")
	log.Printf("📦 Création d’adresse pour user_id=%v", userID)

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "non authentifié"})
		return
	}

	col := database.MongoAddressesDB.Collection("addresses")

	var input models.Address
	if err := c.ShouldBindJSON(&input); err != nil {
		log.Println("❌ Erreur de binding JSON:", err)
		c.JSON(http.StatusBadRequest, gin.H{"message": "Données invalides"})
		return
	}

	// Valeur par défaut si le front ne précise pas le type
	if input.Type == "" {
		input.Type = "user"
	}

	input.ID = primitive.NewObjectID()
	input.UserID = userID
	input.IsDefault = false

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := col.InsertOne(ctx, input)
	if err != nil {
		log.Println("❌ Erreur Mongo InsertOne:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Impossible d'ajouter l'adresse"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Adresse créée",
		"address": input,
	})
}

// 🟢 POST /api/addresses/:id/default
func MakeDefaultAddress(c *gin.Context) {
	idParam := c.Param("id")
	userID := c.GetString("user_id")
	col := database.MongoAddressesDB.Collection("addresses")

	objectID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "ID invalide"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Désactiver tous les autres
	_, _ = col.UpdateMany(ctx, bson.M{"userId": userID}, bson.M{"$set": bson.M{"isDefault": false}})

	// Activer celui-ci
	result, err := col.UpdateOne(ctx,
		bson.M{"_id": objectID, "userId": userID},
		bson.M{"$set": bson.M{"isDefault": true}},
	)

	if err != nil {
		log.Println("❌ Erreur UpdateOne:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Impossible de définir par défaut"})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "Adresse non trouvée"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Adresse mise par défaut", "id": idParam})
}

// 🟢 DELETE /api/addresses/:id
func DeleteAddress(c *gin.Context) {
	idParam := c.Param("id")
	userID := c.GetString("user_id")
	col := database.MongoAddressesDB.Collection("addresses")

	objectID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "ID invalide"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := col.DeleteOne(ctx, bson.M{"_id": objectID, "userId": userID})
	if err != nil {
		log.Println("❌ Erreur DeleteOne:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Suppression impossible"})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "Adresse non trouvée"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Adresse supprimée"})
}
