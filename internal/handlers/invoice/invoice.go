package invoice

import (
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
	"cedra_back_end/internal/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

// POST /api/invoice/:id/send
func SendInvoice(c *gin.Context) {
	id := c.Param("id")
	orderUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalide"})
		return
	}

	session, err := database.GetOrdersSession()
	if err != nil {
		log.Printf("❌ Erreur session ScyllaDB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db non initialisée"})
		return
	}

	// Récupérer la commande
	var (
		userID, paymentIntentID, itemsJSON string
		totalPrice                         float64
		status                             string
		createdAt                          time.Time
		updatedAt                          *time.Time
	)

	err = session.Query(`SELECT user_id, payment_intent_id, items, total_price, status, created_at, updated_at 
	                     FROM orders WHERE order_id = ?`, gocql.UUID(orderUUID)).Scan(
		&userID, &paymentIntentID, &itemsJSON, &totalPrice, &status, &createdAt, &updatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "commande introuvable"})
		return
	}

	// Désérialiser les items
	var items []models.OrderItem
	if itemsJSON != "" {
		if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
			log.Printf("⚠️ Erreur désérialisation items: %v", err)
			items = []models.OrderItem{}
		}
	}

	order := models.Order{
		ID:              gocql.UUID(orderUUID),
		UserID:          userID,
		PaymentIntentID: paymentIntentID,
		Items:           items,
		TotalPrice:      totalPrice,
		Status:          status,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}

	// Récupérer l'email de l'utilisateur depuis la table users
	userSession, err := database.GetUsersSession()
	var userEmail string
	if err == nil {
		userUID, err := uuid.Parse(userID)
		if err == nil {
			err = userSession.Query("SELECT email FROM users WHERE user_id = ?", gocql.UUID(userUID)).Scan(&userEmail)
			if err != nil {
				log.Printf("⚠️ Impossible de récupérer l'email: %v", err)
			}
		}
	}

	if userEmail == "" {
		userEmail = "client@inconnu.tld"
	}

	// 1. Générer le QR SEPA
	iban := os.Getenv("COMPANY_IBAN")
	if iban == "" {
		iban = "BE12345678901234"
	}
	bic := os.Getenv("COMPANY_BIC")
	if bic == "" {
		bic = "KREDBEBB"
	}
	companyName := os.Getenv("COMPANY_NAME")
	if companyName == "" {
		companyName = "Cedra SRL"
	}
	ref := fmt.Sprintf("FACT-%s", id)

	qrBase64, err := utils.GenerateSepaQR(iban, bic, companyName, ref, order.TotalPrice)
	if err != nil {
		log.Println("❌ erreur QR:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "qr fail"})
		return
	}

	// 2. Rendre la page React → PDF
	frontURL := utils.GetFrontendInvoiceBaseURL()
	pdfBytes, err := utils.RenderReactInvoicePDF(frontURL, id, qrBase64)
	if err != nil {
		log.Println("❌ erreur PDF:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "pdf fail"})
		return
	}

	// 3. Corps HTML (tu as déjà un générateur d'e-mail de confirmation)
	htmlBody := utils.GenerateOrderConfirmationHTML(order, userEmail)

	// 4. Envoi
	if err := utils.SendConfirmationEmail(userEmail, "Votre facture Cedra", htmlBody, pdfBytes); err != nil {
		log.Println("❌ erreur envoi mail:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "mail fail"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "facture envoyée",
	})
}
