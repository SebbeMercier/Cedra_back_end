package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log"

	"github.com/gin-gonic/gin"

	"cedra_back_end/internal/utils"
)

// AuditPriceChanges middleware pour auditer les changements de prix
func AuditPriceChanges() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Capturer le body de la requête
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Next()
			return
		}

		// Restaurer le body pour les handlers suivants
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Parser le JSON pour vérifier s'il y a un changement de prix
		var requestData map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &requestData); err != nil {
			c.Next()
			return
		}

		// Vérifier si le prix est modifié
		if price, exists := requestData["price"]; exists {
			productID := c.Param("id")

			// Récupérer l'ancien prix avant la modification
			oldPrice, err := getProductPrice(productID)
			if err != nil {
				log.Printf("⚠️ Erreur récupération ancien prix: %v", err)
			}

			// Stocker les infos pour l'audit post-traitement
			c.Set("audit_price_change", true)
			c.Set("audit_product_id", productID)
			c.Set("audit_old_price", oldPrice)
			c.Set("audit_new_price", price)
		}

		c.Next()

		// Après traitement, enregistrer l'audit si nécessaire
		if shouldAudit, exists := c.Get("audit_price_change"); exists && shouldAudit.(bool) {
			productID, _ := c.Get("audit_product_id")
			oldPrice, _ := c.Get("audit_old_price")
			newPrice, _ := c.Get("audit_new_price")

			// Vérifier que la requête a réussi (status 2xx)
			if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
				oldValue := map[string]interface{}{"price": oldPrice}
				newValue := map[string]interface{}{"price": newPrice}

				utils.LogAction(c, utils.ACTION_PRODUCT_PRICE_CHANGE, utils.RESOURCE_PRODUCT,
					productID.(string), oldValue, newValue)

				log.Printf("💰 Changement de prix audité: produit %s (%.2f → %.2f)",
					productID, oldPrice, newPrice)
			}
		}
	}
}

// AuditCriticalActions middleware pour auditer toutes les actions critiques
func AuditCriticalActions(action, resource string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Capturer les données avant traitement
		resourceID := c.Param("id")
		if resourceID == "" {
			resourceID = c.Param("user_id")
		}
		if resourceID == "" {
			resourceID = c.Param("coupon_id")
		}

		c.Next()

		// Auditer après traitement si succès
		if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
			utils.LogAction(c, action, resource, resourceID, nil, nil)
		} else {
			utils.LogFailedAction(c, action, resource, resourceID, "Action échouée")
		}
	}
}

// getProductPrice récupère le prix actuel d'un produit
func getProductPrice(productID string) (float64, error) {
	// Cette fonction devrait récupérer le prix depuis la base de données
	// Pour l'instant, on retourne 0 en cas d'erreur
	return 0.0, nil
}
