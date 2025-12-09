package product

import (
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

// SearchProductsAdvanced recherche avancée avec filtres et tri
func SearchProductsAdvanced(c *gin.Context) {
	query := c.Query("q")
	categoryID := c.Query("category")
	minPrice := c.Query("min_price")
	maxPrice := c.Query("max_price")
	sortBy := c.DefaultQuery("sort", "relevance")
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "20")

	pageNum, _ := strconv.Atoi(page)
	limitNum, _ := strconv.Atoi(limit)

	if pageNum < 1 {
		pageNum = 1
	}
	if limitNum < 1 || limitNum > 100 {
		limitNum = 20
	}

	session, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	var products []models.Product

	if categoryID != "" {
		catUUID, err := uuid.Parse(categoryID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID catégorie invalide"})
			return
		}

		iter := session.Query(`
			SELECT product_id, name, description, price, stock, category_id, image_urls, tags, created_at, updated_at
			FROM products WHERE category_id = ? ALLOW FILTERING
		`, gocql.UUID(catUUID)).Iter()

		var product models.Product
		for iter.Scan(&product.ID, &product.Name, &product.Description, &product.Price,
			&product.Stock, &product.CategoryID, &product.ImageURLs, &product.Tags,
			&product.CreatedAt, &product.UpdatedAt) {
			products = append(products, product)
		}
		iter.Close()
	} else {
		iter := session.Query(`
			SELECT product_id, name, description, price, stock, category_id, image_urls, tags, created_at, updated_at
			FROM products
		`).Iter()

		var product models.Product
		for iter.Scan(&product.ID, &product.Name, &product.Description, &product.Price,
			&product.Stock, &product.CategoryID, &product.ImageURLs, &product.Tags,
			&product.CreatedAt, &product.UpdatedAt) {
			products = append(products, product)
		}
		iter.Close()
	}

	// Filtrer par prix
	if minPrice != "" || maxPrice != "" {
		var minPriceFloat, maxPriceFloat float64
		if minPrice != "" {
			minPriceFloat, _ = strconv.ParseFloat(minPrice, 64)
		}
		if maxPrice != "" {
			maxPriceFloat, _ = strconv.ParseFloat(maxPrice, 64)
		}

		var filtered []models.Product
		for _, p := range products {
			if minPrice != "" && p.Price < minPriceFloat {
				continue
			}
			if maxPrice != "" && p.Price > maxPriceFloat {
				continue
			}
			filtered = append(filtered, p)
		}
		products = filtered
	}

	// Filtrer par query
	if query != "" {
		var filtered []models.Product
		queryLower := strings.ToLower(query)
		for _, p := range products {
			if strings.Contains(strings.ToLower(p.Name), queryLower) ||
				strings.Contains(strings.ToLower(p.Description), queryLower) {
				filtered = append(filtered, p)
			}
		}
		products = filtered
	}

	// Trier
	switch sortBy {
	case "price_asc":
		sortByPriceAsc(products)
	case "price_desc":
		sortByPriceDesc(products)
	case "newest":
		sortByNewest(products)
	}

	// Pagination
	total := len(products)
	start := (pageNum - 1) * limitNum
	end := start + limitNum

	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	paginatedProducts := products[start:end]

	c.JSON(http.StatusOK, gin.H{
		"products": paginatedProducts,
		"pagination": gin.H{
			"page":        pageNum,
			"limit":       limitNum,
			"total":       total,
			"total_pages": (total + limitNum - 1) / limitNum,
		},
		"filters": gin.H{
			"query":     query,
			"category":  categoryID,
			"min_price": minPrice,
			"max_price": maxPrice,
			"sort":      sortBy,
		},
	})
}

// GetProductFilters retourne les filtres disponibles
func GetProductFilters(c *gin.Context) {
	session, err := database.GetProductsSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	categoriesIter := session.Query("SELECT category_id, name FROM categories").Iter()

	type CategoryFilter struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	var categories []CategoryFilter
	var cat CategoryFilter
	var catID gocql.UUID

	for categoriesIter.Scan(&catID, &cat.Name) {
		cat.ID = catID.String()
		categories = append(categories, cat)
	}
	categoriesIter.Close()

	var minPrice, maxPrice float64
	productsIter := session.Query("SELECT price FROM products").Iter()
	var price float64
	first := true

	for productsIter.Scan(&price) {
		if first {
			minPrice = price
			maxPrice = price
			first = false
		} else {
			if price < minPrice {
				minPrice = price
			}
			if price > maxPrice {
				maxPrice = price
			}
		}
	}
	productsIter.Close()

	c.JSON(http.StatusOK, gin.H{
		"categories": categories,
		"price_range": gin.H{
			"min": minPrice,
			"max": maxPrice,
		},
		"sort_options": []gin.H{
			{"value": "relevance", "label": "Pertinence"},
			{"value": "price_asc", "label": "Prix croissant"},
			{"value": "price_desc", "label": "Prix décroissant"},
			{"value": "newest", "label": "Plus récents"},
		},
	})
}

func sortByPriceAsc(products []models.Product) {
	for i := 0; i < len(products)-1; i++ {
		for j := i + 1; j < len(products); j++ {
			if products[i].Price > products[j].Price {
				products[i], products[j] = products[j], products[i]
			}
		}
	}
}

func sortByPriceDesc(products []models.Product) {
	for i := 0; i < len(products)-1; i++ {
		for j := i + 1; j < len(products); j++ {
			if products[i].Price < products[j].Price {
				products[i], products[j] = products[j], products[i]
			}
		}
	}
}

func sortByNewest(products []models.Product) {
	for i := 0; i < len(products)-1; i++ {
		for j := i + 1; j < len(products); j++ {
			if products[i].CreatedAt.Before(products[j].CreatedAt) {
				products[i], products[j] = products[j], products[i]
			}
		}
	}
}
