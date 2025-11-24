package product

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/google/uuid"

	"cedra_back_end/internal/database"
)

func UpdateProduct(c *gin.Context) {
	productID := c.Param("id")

	productUUID, err := uuid.Parse(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID produit invalide"})
		return
	}

	var input struct {
		Name        *string   `json:"name"`
		Description *string   `json:"description"`
		Price       *float64  `json:"price"`
		Stock       *int      `json:"stock"`
		CategoryID  *string   `json:"category_id"`
		Tags        *[]string `json:"tags"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides"})
		return
	}

	session, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	updates := []string{}
	values := []interface{}{}

	if input.Name != nil {
		updates = append(updates, "name = ?")
		values = append(values, *input.Name)
	}
	if input.Description != nil {
		updates = append(updates, "description = ?")
		values = append(values, *input.Description)
	}
	if input.Price != nil {
		updates = append(updates, "price = ?")
		values = append(values, *input.Price)
	}
	if input.Stock != nil {
		updates = append(updates, "stock = ?")
		values = append(values, *input.Stock)
	}
	if input.CategoryID != nil {
		catUUID, err := uuid.Parse(*input.CategoryID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Category ID invalide"})
			return
		}
		updates = append(updates, "category_id = ?")
		values = append(values, gocql.UUID(catUUID))
	}
	if input.Tags != nil {
		updates = append(updates, "tags = ?")
		values = append(values, *input.Tags)
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Aucune donnée à mettre à jour"})
		return
	}

	updates = append(updates, "updated_at = ?")
	now := time.Now()
	values = append(values, now)

	// Ajouter product_id à la fin
	values = append(values, gocql.UUID(productUUID))

	query := "UPDATE products SET " + updates[0]
	for i := 1; i < len(updates); i++ {
		query += ", " + updates[i]
	}
	query += " WHERE product_id = ?"

	if err := session.Query(query, values...).Exec(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	// 🔹 Invalider le cache Redis
	ctx := context.Background()
	cacheKey := "product:full:" + productID
	database.RedisClient.Del(ctx, cacheKey)

	c.JSON(http.StatusOK, gin.H{"message": "Produit mis à jour avec succès"})
}

func DeleteProduct(c *gin.Context) {
	productID := c.Param("id")

	productUUID, err := uuid.Parse(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID produit invalide"})
		return
	}

	session, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	// 🔹 Supprimer le produit
	if err := session.Query("DELETE FROM products WHERE product_id = ?", gocql.UUID(productUUID)).Exec(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression"})
		return
	}

	// 🔹 Invalider le cache Redis
	ctx := context.Background()
	cacheKey := "product:full:" + productID
	database.RedisClient.Del(ctx, cacheKey)

	c.JSON(http.StatusOK, gin.H{"message": "Produit supprimé avec succès"})
}
