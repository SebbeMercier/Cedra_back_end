package product

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"

	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
)

// CreateProductVariant - Créer une variante de produit
func CreateProductVariant(c *gin.Context) {
	productIDStr := c.Param("id")

	var req struct {
		SKU        string            `json:"sku" binding:"required"`
		Price      float64           `json:"price" binding:"required"`
		Stock      int               `json:"stock" binding:"required"`
		Attributes map[string]string `json:"attributes" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides: " + err.Error()})
		return
	}

	productID, err := gocql.ParseUUID(productIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID produit invalide"})
		return
	}

	// Vérifier que le produit existe
	productsSession, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	checkQuery := `SELECT product_id FROM ks_products.products WHERE product_id = ?`
	var tempProductID gocql.UUID
	if err := productsSession.Query(checkQuery, productID).Scan(&tempProductID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produit non trouvé"})
		return
	}

	// Vérifier que le SKU n'existe pas déjà
	var existingSKU string
	skuQuery := `SELECT sku FROM ks_products.product_variants WHERE sku = ? LIMIT 1`
	if err := productsSession.Query(skuQuery, req.SKU).Scan(&existingSKU); err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Ce SKU existe déjà"})
		return
	}

	variant := models.ProductVariant{
		ID:         gocql.TimeUUID(),
		ProductID:  productID,
		SKU:        req.SKU,
		Price:      req.Price,
		Stock:      req.Stock,
		Attributes: req.Attributes,
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Insérer la variante
	insertQuery := `
		INSERT INTO ks_products.product_variants (
			id, product_id, sku, price, stock, attributes, is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	if err := productsSession.Query(insertQuery,
		variant.ID, variant.ProductID, variant.SKU, variant.Price, variant.Stock,
		variant.Attributes, variant.IsActive, variant.CreatedAt, variant.UpdatedAt,
	).Exec(); err != nil {
		log.Printf("❌ Erreur création variante: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création de la variante"})
		return
	}

	// Marquer le produit comme ayant des variantes
	updateProductQuery := `UPDATE ks_products.products SET has_variants = true, updated_at = ? WHERE product_id = ?`
	if err := productsSession.Query(updateProductQuery, time.Now(), productID).Exec(); err != nil {
		log.Printf("⚠️ Erreur mise à jour has_variants: %v", err)
	}

	log.Printf("✅ Variante créée: %s pour produit %s", variant.SKU, productID)
	c.JSON(http.StatusCreated, gin.H{
		"message": "Variante créée avec succès",
		"variant": variant,
	})
}

// GetProductVariants - Récupérer les variantes d'un produit
func GetProductVariants(c *gin.Context) {
	productIDStr := c.Param("id")

	productID, err := gocql.ParseUUID(productIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID produit invalide"})
		return
	}

	query := `SELECT id, product_id, sku, price, stock, attributes, is_active, created_at, updated_at 
			  FROM ks_products.product_variants WHERE product_id = ? AND is_active = true`

	productsSession, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	iter := productsSession.Query(query, productID).Iter()
	defer iter.Close()

	var variants []models.ProductVariant
	var variant models.ProductVariant

	for iter.Scan(&variant.ID, &variant.ProductID, &variant.SKU, &variant.Price,
		&variant.Stock, &variant.Attributes, &variant.IsActive, &variant.CreatedAt,
		&variant.UpdatedAt) {
		variants = append(variants, variant)
	}

	if err := iter.Close(); err != nil {
		log.Printf("❌ Erreur récupération variantes: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur serveur"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"variants": variants,
		"total":    len(variants),
	})
}

// UpdateProductVariant - Mettre à jour une variante
func UpdateProductVariant(c *gin.Context) {
	variantIDStr := c.Param("variant_id")

	var req struct {
		Price      *float64           `json:"price"`
		Stock      *int               `json:"stock"`
		Attributes *map[string]string `json:"attributes"`
		IsActive   *bool              `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides"})
		return
	}

	variantID, err := gocql.ParseUUID(variantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID variante invalide"})
		return
	}

	// Construire la requête de mise à jour dynamiquement
	updates := []string{}
	values := []interface{}{}

	if req.Price != nil {
		updates = append(updates, "price = ?")
		values = append(values, *req.Price)
	}

	if req.Stock != nil {
		updates = append(updates, "stock = ?")
		values = append(values, *req.Stock)
	}

	if req.Attributes != nil {
		updates = append(updates, "attributes = ?")
		values = append(values, *req.Attributes)
	}

	if req.IsActive != nil {
		updates = append(updates, "is_active = ?")
		values = append(values, *req.IsActive)
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Aucune mise à jour fournie"})
		return
	}

	updates = append(updates, "updated_at = ?")
	values = append(values, time.Now())
	values = append(values, variantID)

	query := `UPDATE ks_products.product_variants SET ` +
		updates[0]
	for i := 1; i < len(updates); i++ {
		query += ", " + updates[i]
	}
	query += " WHERE id = ?"

	productsSession, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	if err := productsSession.Query(query, values...).Exec(); err != nil {
		log.Printf("❌ Erreur mise à jour variante: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Variante mise à jour avec succès"})
}

// DeleteProductVariant - Supprimer une variante
func DeleteProductVariant(c *gin.Context) {
	variantIDStr := c.Param("variant_id")

	variantID, err := gocql.ParseUUID(variantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID variante invalide"})
		return
	}

	// Marquer comme inactive plutôt que supprimer
	query := `UPDATE ks_products.product_variants SET is_active = false, updated_at = ? WHERE id = ?`
	productsSession, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	if err := productsSession.Query(query, time.Now(), variantID).Exec(); err != nil {
		log.Printf("❌ Erreur suppression variante: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Variante supprimée avec succès"})
}

// GetVariantBySKU - Récupérer une variante par SKU
func GetVariantBySKU(c *gin.Context) {
	sku := c.Param("sku")

	var variant models.ProductVariant
	query := `SELECT id, product_id, sku, price, stock, attributes, is_active, created_at, updated_at 
			  FROM ks_products.product_variants WHERE sku = ? AND is_active = true LIMIT 1`

	productsSession, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	if err := productsSession.Query(query, sku).Scan(
		&variant.ID, &variant.ProductID, &variant.SKU, &variant.Price,
		&variant.Stock, &variant.Attributes, &variant.IsActive,
		&variant.CreatedAt, &variant.UpdatedAt,
	); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Variante non trouvée"})
		return
	}

	c.JSON(http.StatusOK, variant)
}
