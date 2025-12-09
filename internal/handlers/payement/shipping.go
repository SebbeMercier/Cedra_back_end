package pa

import (
	"cedra_back_end/internal/models"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetShippingOptions retourne les options de livraison disponibles
func GetShippingOptions(c *gin.Context) {
	// Récupérer le total du panier depuis le query param
	var cartTotal float64
	if total := c.Query("cart_total"); total != "" {
		if _, err := c.GetQuery("cart_total"); err {
			// Parse le total
			if n, err := parseFloat(total); err == nil {
				cartTotal = n
			}
		}
	}

	// Seuil de livraison gratuite
	freeThreshold := 50.0
	isFree := cartTotal >= freeThreshold

	options := []models.ShippingOption{
		{
			ID:            "standard",
			Name:          "Livraison Standard",
			Description:   "Livraison en 5-7 jours ouvrés",
			Price:         5.99,
			EstimatedDays: 7,
		},
		{
			ID:            "express",
			Name:          "Livraison Express",
			Description:   "Livraison en 2-3 jours ouvrés",
			Price:         12.99,
			EstimatedDays: 3,
		},
		{
			ID:            "next_day",
			Name:          "Livraison 24h",
			Description:   "Livraison le lendemain avant 18h",
			Price:         19.99,
			EstimatedDays: 1,
		},
	}

	// Si livraison gratuite, mettre le prix à 0 pour l'option standard
	if isFree {
		options[0].Price = 0
		options[0].Name = "Livraison Standard Gratuite"
	}

	calculation := models.ShippingCalculation{
		Options:       options,
		FreeThreshold: freeThreshold,
		CartTotal:     cartTotal,
		IsFree:        isFree,
	}

	c.JSON(http.StatusOK, calculation)
}

// Helper function
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
