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

// =========================
// 🟢 CRÉER UNE CATÉGORIE
// =========================
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

	// Insertion dans categories
	err = session.Query(
		`INSERT INTO categories (category_id, name, slug, description, parent_category_id, image_url, created_at) 
        VALUES (?, ?, ?, ?, ?, ?, ?)`,
		categoryID, cat.Name, cat.Slug, cat.Description, cat.ParentCategoryID, cat.ImageURL, now,
	).Exec()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur création catégorie: " + err.Error()})
		return
	}

	// Invalider le cache Redis
	ctx := context.Background()
	database.RedisClient.Del(ctx, "categories:all")

	c.JSON(http.StatusCreated, gin.H{
		"message": "✅ Catégorie créée avec succès",
		"id":      categoryID.String(),
		"name":    cat.Name,
	})
}

// =========================
// 🔵 LISTER TOUTES LES CATÉGORIES
// =========================
func GetAllCategories(c *gin.Context) {
	ctx := context.Background()
	cacheKey := "categories:all"

	// 1️⃣ Vérifier le cache Redis
	val, err := database.RedisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		var cached []models.Category
		if json.Unmarshal([]byte(val), &cached) == nil {
			c.JSON(http.StatusOK, cached)
			return
		}
	}

	// 2️⃣ Récupérer depuis ScyllaDB
	session, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	var cats []models.Category
	iter := session.Query(
		"SELECT category_id, name, slug, description, parent_category_id, image_url, created_at FROM categories",
	).Iter()

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lecture catégories: " + err.Error()})
		return
	}

	// 3️⃣ Mettre en cache
	data, _ := json.Marshal(cats)
	database.RedisClient.Set(ctx, cacheKey, data, 1*time.Hour)

	c.JSON(http.StatusOK, cats)
}

// =========================
// 🟡 RÉCUPÉRER UNE CATÉGORIE PAR ID
// =========================
func GetCategoryByID(c *gin.Context) {
	categoryID := c.Param("id")

	uuid, err := gocql.ParseUUID(categoryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID catégorie invalide"})
		return
	}

	session, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	var (
		name, slug, description, imageURL string
		parentCategoryID                  *gocql.UUID
		createdAt                         time.Time
	)

	err = session.Query(
		"SELECT name, slug, description, parent_category_id, image_url, created_at FROM categories WHERE category_id = ?",
		uuid,
	).Scan(&name, &slug, &description, &parentCategoryID, &imageURL, &createdAt)

	if err != nil {
		if err == gocql.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Catégorie introuvable"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	cat := models.Category{
		ID:               uuid,
		Name:             name,
		Slug:             slug,
		Description:      description,
		ParentCategoryID: parentCategoryID,
		ImageURL:         imageURL,
		CreatedAt:        &createdAt,
	}

	c.JSON(http.StatusOK, cat)
}
