package pa

import "cedra_back_end/internal/models"

// calcTotal calcule le montant total d'un panier
func calcTotal(items []models.CartItem) float64 {
	var total float64
	for _, item := range items {
		total += item.Price * float64(item.Quantity)
	}
	return total
}
