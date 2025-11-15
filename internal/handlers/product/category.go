package product

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
)

// 🟢 Créer une catégorie
func CreateCategory(c *gin.Context) {
	session, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	var cat models.Category
	if err := c.ShouldBindJSON(&cat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if cat.Name == "" || cat.Slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Les champs 'name' et 'slug' sont obligatoires"})
		return
	}

	// Générer un UUID pour la catégorie
	categoryID := gocql.TimeUUID()
	cat.ID = categoryID
	now := time.Now()
	cat.CreatedAt = &now

	// Insert dans categories
	err = session.Query(`INSERT INTO categories (category_id, name, slug, description, parent_category_id, image_url, created_at) 
	                     VALUES (?, ?, ?, ?, ?, ?, ?)`,
		categoryID, cat.Name, cat.Slug, cat.Description, cat.ParentCategoryID, cat.ImageURL, now).Exec()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": categoryID.String()})
}

// 🔵 Lister les catégories
func GetAllCategories(c *gin.Context) {
	ctx := context.Background()
	cacheKey := "categories:all"

	// Cache Redis
	val, err := database.RedisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		var cached []models.Category
		json.Unmarshal([]byte(val), &cached)
		c.JSON(http.StatusOK, cached)
		return
	}

	session, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	// Récupérer toutes les catégories
	var cats []models.Category
	iter := session.Query("SELECT category_id, name, slug, description, parent_category_id, image_url, created_at FROM categories").Iter()
	var (
		categoryID                        gocql.UUID
		name, slug, description, imageURL string
		parentCategoryID                  *gocql.UUID
		createdAt                         time.Time
	)
	for iter.Scan(&categoryID, &name, &slug, &description, &parentCategoryID, &imageURL, &createdAt) {
		cat := models.Category{
			ID:               categoryID,
			Name:             name,
			Slug:             slug,
			Description:      description,
			ParentCategoryID: parentCategoryID,
			ImageURL:         imageURL,
			CreatedAt:        &createdAt,
		}
		cats = append(cats, cat)
	}
	if err := iter.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Mettre en cache
	data, _ := json.Marshal(cats)
	database.RedisClient.Set(ctx, cacheKey, data, time.Hour)

	c.JSON(http.StatusOK, cats)
}
