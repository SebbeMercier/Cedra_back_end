package user

import (
	"cedra_back_end/internal/database"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"golang.org/x/crypto/bcrypt"
)

// DeleteAccount supprime complètement un compte utilisateur et toutes ses données associées
func DeleteAccount(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur non authentifié"})
		return
	}

	var input struct {
		Password        string `json:"password"`        // Pour confirmer l'identité (auth locale)
		ConfirmDeletion bool   `json:"confirmDeletion"` // Confirmation explicite
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides"})
		return
	}

	if !input.ConfirmDeletion {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vous devez confirmer la suppression"})
		return
	}

	id := fmt.Sprintf("%v", userID)
	uid, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID utilisateur invalide"})
		return
	}
	userUUID := gocql.UUID(uid)

	// =============================================
	// 1. VÉRIFIER L'IDENTITÉ DE L'UTILISATEUR
	// =============================================

	usersSession, err := database.GetUsersSession()
	if err != nil {
		log.Printf("❌ Erreur session ScyllaDB users: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	var (
		email, password, provider string
	)

	err = usersSession.Query(`SELECT email, password, provider FROM users WHERE user_id = ?`, userUUID).
		Scan(&email, &password, &provider)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur introuvable"})
		return
	}

	// Vérifier le mot de passe pour les comptes locaux
	if provider == "local" {
		if input.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Mot de passe requis pour confirmer la suppression"})
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(password), []byte(input.Password)) != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Mot de passe incorrect"})
			return
		}
	}

	log.Printf("🗑️ Début de la suppression du compte: %s (%s)", email, id)

	// =============================================
	// 2. SUPPRIMER LES DONNÉES DANS REDIS (PANIER)
	// =============================================

	ctx := context.Background()
	cartKey := "cart:" + id

	err = database.Redis.Del(ctx, cartKey).Err()
	if err != nil {
		log.Printf("⚠️ Erreur suppression panier Redis: %v", err)
	} else {
		log.Printf("✅ Panier supprimé de Redis")
	}

	// Supprimer les sessions et tokens éventuels
	sessionKeys := []string{
		"session:" + id,
		"oauth_redirect:" + id,
		"reset_token:" + email,
	}
	for _, key := range sessionKeys {
		database.Redis.Del(ctx, key)
	}

	// =============================================
	// 3. SUPPRIMER LES ADRESSES (KEYSPACE USERS)
	// =============================================

	// Récupérer toutes les adresses de l'utilisateur
	iter := usersSession.Query(`SELECT address_id FROM addresses WHERE user_id = ?`, id).Iter()
	var addressID gocql.UUID
	addressCount := 0

	for iter.Scan(&addressID) {
		err = usersSession.Query(`DELETE FROM addresses WHERE address_id = ?`, addressID).Exec()
		if err != nil {
			log.Printf("⚠️ Erreur suppression adresse %s: %v", addressID, err)
		} else {
			addressCount++
		}
	}
	iter.Close()
	log.Printf("✅ %d adresse(s) supprimée(s)", addressCount)

	// =============================================
	// 4. SUPPRIMER LES COMMANDES (KEYSPACE ORDERS)
	// =============================================

	ordersSession, err := database.GetOrdersSession()
	if err != nil {
		log.Printf("⚠️ Erreur session ScyllaDB orders: %v", err)
	} else {
		// Supprimer de orders_by_user
		err = ordersSession.Query(`DELETE FROM orders_by_user WHERE user_id = ?`, id).Exec()
		if err != nil {
			log.Printf("⚠️ Erreur suppression orders_by_user: %v", err)
		} else {
			log.Printf("✅ Index orders_by_user supprimé")
		}

		// Récupérer et supprimer toutes les commandes principales
		iter := ordersSession.Query(`SELECT order_id FROM orders WHERE user_id = ?`, id).Iter()
		var orderID gocql.UUID
		orderCount := 0

		for iter.Scan(&orderID) {
			err = ordersSession.Query(`DELETE FROM orders WHERE order_id = ?`, orderID).Exec()
			if err != nil {
				log.Printf("⚠️ Erreur suppression commande %s: %v", orderID, err)
			} else {
				orderCount++
			}
		}
		iter.Close()
		log.Printf("✅ %d commande(s) supprimée(s)", orderCount)
	}

	// =============================================
	// 5. SUPPRIMER LES PRODUITS SI VENDEUR (KEYSPACE PRODUCTS)
	// =============================================

	// Note: Adapter selon votre logique métier
	// Si l'utilisateur a une entreprise, on pourrait supprimer ses produits
	// Mais attention : si plusieurs utilisateurs partagent la même entreprise,
	// il ne faut pas supprimer les produits !
	log.Printf("ℹ️ Suppression des produits non implémentée (nécessite logique métier)")

	// =============================================
	// 6. SUPPRIMER LES IMAGES MINIO
	// =============================================

	// Supprimer les images de profil ou autres fichiers associés
	bucketName := "cedra-images"
	userPrefix := "users/" + id + "/"

	objectsCh := database.MinIO.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Prefix:    userPrefix,
		Recursive: true,
	})

	imageCount := 0
	for object := range objectsCh {
		if object.Err != nil {
			log.Printf("⚠️ Erreur listage MinIO: %v", object.Err)
			continue
		}
		err = database.MinIO.RemoveObject(ctx, bucketName, object.Key, minio.RemoveObjectOptions{})
		if err != nil {
			log.Printf("⚠️ Erreur suppression image %s: %v", object.Key, err)
		} else {
			imageCount++
		}
	}
	log.Printf("✅ %d image(s) supprimée(s) de MinIO", imageCount)

	// =============================================
	// 7. SUPPRIMER DE ELASTICSEARCH
	// =============================================

	// Supprimer l'utilisateur de l'index Elasticsearch si indexé
	// Note: Adapter selon vos index Elasticsearch
	if database.Elastic != nil {
		_, err = database.Elastic.Delete("users", id)
		if err != nil {
			log.Printf("⚠️ Erreur suppression Elasticsearch: %v", err)
		} else {
			log.Printf("✅ Utilisateur supprimé d'Elasticsearch")
		}
	}

	// =============================================
	// 8. SUPPRIMER L'UTILISATEUR (KEYSPACE USERS)
	// =============================================

	// Supprimer de users_by_email (index)
	err = usersSession.Query(`DELETE FROM users_by_email WHERE email = ?`, email).Exec()
	if err != nil {
		log.Printf("⚠️ Erreur suppression users_by_email: %v", err)
	} else {
		log.Printf("✅ Index users_by_email supprimé")
	}

	// Supprimer l'utilisateur principal
	err = usersSession.Query(`DELETE FROM users WHERE user_id = ?`, userUUID).Exec()
	if err != nil {
		log.Printf("❌ Erreur suppression utilisateur: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression du compte"})
		return
	}

	log.Printf("✅ Utilisateur %s (%s) complètement supprimé", email, id)

	c.JSON(http.StatusOK, gin.H{
		"message":    "Votre compte et toutes vos données ont été supprimés définitivement",
		"deleted_at": time.Now().Format(time.RFC3339),
	})
}
