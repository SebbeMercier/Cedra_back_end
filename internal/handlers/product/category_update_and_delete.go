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

func UpdateCategory(c *gin.Context) {
	categoryID := c.Param("id")

	categoryUUID, err := uuid.Parse(categoryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID catégorie invalide"})
		return
	}

	var input struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
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

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Aucune donnée à mettre à jour"})
		return
	}

	updates = append(updates, "updated_at = ?")
	now := time.Now()
	values = append(values, now)
	values = append(values, gocql.UUID(categoryUUID))

	query := "UPDATE categories SET " + updates[0]
	for i := 1; i < len(updates); i++ {
		query += ", " + updates[i]
	}
	query += " WHERE category_id = ?"

	if err := session.Query(query, values...).Exec(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	// 🔹 Invalider le cache Redis
	ctx := context.Background()
	database.RedisClient.Del(ctx, "categories:all")

	c.JSON(http.StatusOK, gin.H{"message": "Catégorie mise à jour avec succès"})
}

func DeleteCategory(c *gin.Context) {
	categoryID := c.Param("id")

	categoryUUID, err := uuid.Parse(categoryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID catégorie invalide"})
		return
	}

	session, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	// ⚠️ Vérifier qu'aucun produit n'utilise cette catégorie
	var count int
	if err := session.Query("SELECT COUNT(*) FROM products WHERE category_id = ? ALLOW FILTERING", gocql.UUID(categoryUUID)).Scan(&count); err == nil && count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Impossible de supprimer : des produits utilisent cette catégorie"})
		return
	}

	if err := session.Query("DELETE FROM categories WHERE category_id = ?", gocql.UUID(categoryUUID)).Exec(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression"})
		return
	}

	ctx := context.Background()
	database.RedisClient.Del(ctx, "categories:all")

	c.JSON(http.StatusOK, gin.H{"message": "Catégorie supprimée avec succès"})
}
