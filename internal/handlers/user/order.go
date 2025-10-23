package user

import (
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ✅ Récupère toutes les commandes de l'utilisateur connecté
func GetMyOrders(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur non authentifié"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := database.MongoOrdersDB.Collection("orders")
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := collection.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		log.Println("❌ Erreur MongoDB Find:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur récupération commandes"})
		return
	}
	defer cursor.Close(ctx)

	var orders []models.Order
	if err := cursor.All(ctx, &orders); err != nil {
		log.Println("❌ Erreur décodage commandes:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur décodage"})
		return
	}

	// ✅ Enrichir avec les infos produits
	productsCollection := database.MongoProductsDB.Collection("products")
	for i := range orders {
		for j := range orders[i].Items {
			var product models.Product
			err := productsCollection.FindOne(ctx, bson.M{"_id": orders[i].Items[j].ProductID}).Decode(&product)
			if err == nil {
				orders[i].Items[j].ProductName = product.Name // ⚠️ Ajoute ce champ dans ton model OrderItem
			}
		}
	}

	log.Printf("✅ %d commandes trouvées pour user %s", len(orders), userID)

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
	})
}

// ✅ Récupère une commande spécifique par ID
func GetOrderByID(c *gin.Context) {
	userID := c.GetString("user_id")
	orderID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur non authentifié"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := database.MongoOrdersDB.Collection("orders")

	var order models.Order
	err := collection.FindOne(ctx, bson.M{
		"_id":     orderID,
		"user_id": userID, // ✅ Sécurité : on vérifie que la commande appartient bien à l'utilisateur
	}).Decode(&order)

	if err != nil {
		log.Println("❌ Commande introuvable:", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Commande introuvable"})
		return
	}

	c.JSON(http.StatusOK, order)
}