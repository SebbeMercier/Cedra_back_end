package user

import (
	"log"
	"net/http"

	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
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

	session, err := database.GetUsersSession()
	if err != nil {
		log.Printf("❌ Erreur session ScyllaDB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erreur connexion base de données"})
		return
	}

	// 🔍 Recherche toutes les adresses personnelles OU liées à la société
	// Note: ScyllaDB ne supporte pas $or facilement, on fait deux requêtes
	var results []models.Address

	// Adresses personnelles
	iter := session.Query("SELECT address_id, user_id, company_id, street, postal_code, city, country, type, is_default FROM addresses WHERE user_id = ? AND type != ? ALLOW FILTERING", userID, "billing").Iter()
	var (
		addressID gocql.UUID
		userIDDB, companyIDDB, street, postalCode, city, country, typeAddr string
		isDefault bool
	)
	for iter.Scan(&addressID, &userIDDB, &companyIDDB, &street, &postalCode, &city, &country, &typeAddr, &isDefault) {
		var companyIDPtr *string
		if companyIDDB != "" {
			companyIDPtr = &companyIDDB
		}
		results = append(results, models.Address{
			ID:         addressID,
			UserID:     userIDDB,
			CompanyID:  companyIDPtr,
			Street:     street,
			PostalCode: postalCode,
			City:       city,
			Country:    country,
			Type:       typeAddr,
			IsDefault:  isDefault,
		})
	}
	if err := iter.Close(); err != nil {
		log.Printf("⚠️ Erreur fermeture iter: %v", err)
	}

	// Si companyID est fourni, chercher aussi les adresses de la société
	if companyID != "" {
		iter2 := session.Query("SELECT address_id, user_id, company_id, street, postal_code, city, country, type, is_default FROM addresses WHERE company_id = ? AND type != ? ALLOW FILTERING", companyID, "billing").Iter()
		for iter2.Scan(&addressID, &userIDDB, &companyIDDB, &street, &postalCode, &city, &country, &typeAddr, &isDefault) {
			var companyIDPtr *string
			if companyIDDB != "" {
				companyIDPtr = &companyIDDB
			}
			// Éviter les doublons
			found := false
			for _, r := range results {
				if r.ID == addressID {
					found = true
					break
				}
			}
			if !found {
				results = append(results, models.Address{
					ID:         addressID,
					UserID:     userIDDB,
					CompanyID:  companyIDPtr,
					Street:     street,
					PostalCode: postalCode,
					City:       city,
					Country:    country,
					Type:       typeAddr,
					IsDefault:  isDefault,
				})
			}
		}
		iter2.Close()
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

	session, err := database.GetUsersSession()
	if err != nil {
		log.Printf("❌ Erreur session ScyllaDB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erreur connexion base de données"})
		return
	}

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

	addressID := gocql.TimeUUID()
	input.ID = addressID
	input.UserID = userID
	input.IsDefault = false

	var companyIDStr string
	if input.CompanyID != nil {
		companyIDStr = *input.CompanyID
	}

	err = session.Query(`INSERT INTO addresses (address_id, user_id, company_id, street, postal_code, city, country, type, is_default) 
	                     VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		addressID, userID, companyIDStr, input.Street, input.PostalCode, input.City, input.Country, input.Type, false).Exec()
	if err != nil {
		log.Printf("❌ Erreur insertion adresse: %v", err)
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

	session, err := database.GetUsersSession()
	if err != nil {
		log.Printf("❌ Erreur session ScyllaDB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erreur connexion base de données"})
		return
	}

	addressID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "ID invalide"})
		return
	}
	addressUUID := gocql.UUID(addressID)

	// Vérifier que l'adresse appartient à l'utilisateur
	var userIDDB string
	err = session.Query("SELECT user_id FROM addresses WHERE address_id = ?", addressUUID).Scan(&userIDDB)
	if err != nil || userIDDB != userID {
		c.JSON(http.StatusNotFound, gin.H{"message": "Adresse non trouvée"})
		return
	}

	// Désactiver tous les autres (nécessite un scan car ScyllaDB ne supporte pas UPDATE WHERE avec condition)
	iter := session.Query("SELECT address_id FROM addresses WHERE user_id = ? ALLOW FILTERING", userID).Iter()
	var otherID gocql.UUID
	for iter.Scan(&otherID) {
		if otherID != addressUUID {
			session.Query("UPDATE addresses SET is_default = ? WHERE address_id = ?", false, otherID).Exec()
		}
	}
	iter.Close()

	// Activer celui-ci
	err = session.Query("UPDATE addresses SET is_default = ? WHERE address_id = ?", true, addressUUID).Exec()
	if err != nil {
		log.Printf("❌ Erreur UpdateOne: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Impossible de définir par défaut"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Adresse mise par défaut", "id": idParam})
}

// 🟢 DELETE /api/addresses/:id
func DeleteAddress(c *gin.Context) {
	idParam := c.Param("id")
	userID := c.GetString("user_id")

	session, err := database.GetUsersSession()
	if err != nil {
		log.Printf("❌ Erreur session ScyllaDB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erreur connexion base de données"})
		return
	}

	addressID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "ID invalide"})
		return
	}
	addressUUID := gocql.UUID(addressID)

	// Vérifier que l'adresse appartient à l'utilisateur
	var userIDDB string
	err = session.Query("SELECT user_id FROM addresses WHERE address_id = ?", addressUUID).Scan(&userIDDB)
	if err != nil || userIDDB != userID {
		c.JSON(http.StatusNotFound, gin.H{"message": "Adresse non trouvée"})
		return
	}

	// Supprimer l'adresse
	err = session.Query("DELETE FROM addresses WHERE address_id = ?", addressUUID).Exec()
	if err != nil {
		log.Printf("❌ Erreur DeleteOne: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Suppression impossible"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Adresse supprimée"})
}
