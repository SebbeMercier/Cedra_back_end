package utils

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"cedra_back_end/internal/models"

	"github.com/wneessen/go-mail"
)

func SendConfirmationEmail(to, subject, htmlBody string, pdfAttachment []byte) error {
	msg := mail.NewMsg()

	if err := msg.From("noreply@eldocam.com"); err != nil {
		return err
	}
	if err := msg.To(to); err != nil {
		return err
	}
	msg.Subject(subject)
	msg.SetBodyString(mail.TypeTextHTML, htmlBody)

	// ✅ FIX : Utilise AttachReader au lieu de AddAttachment
	if pdfAttachment != nil {
		msg.AttachReader("facture_cedra.pdf", bytes.NewReader(pdfAttachment))
	}

	client, err := mail.NewClient("ssl0.ovh.net",
		mail.WithPort(587),
		mail.WithSMTPAuth(mail.SMTPAuthLogin),
		mail.WithUsername(os.Getenv("SMTP_USERNAME")),
		mail.WithPassword(os.Getenv("SMTP_PASSWORD")),
		mail.WithTLSPolicy(mail.TLSMandatory),
	)
	if err != nil {
		return err
	}

	log.Println("📤 Envoi de l'e-mail à", to)
	return client.DialAndSend(msg)
}

// GenerateOrderConfirmationHTML génère le HTML de confirmation de commande
func GenerateOrderConfirmationHTML(order models.Order, userEmail string) string {
	itemsHTML := ""
	for _, item := range order.Items {
		itemsHTML += fmt.Sprintf(`
			<tr>
				<td>%s</td>
				<td>%d</td>
				<td>%.2f€</td>
				<td>%.2f€</td>
			</tr>`, item.Name, item.Quantity, item.Price, item.Price*float64(item.Quantity))
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="fr">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Confirmation de commande</title>
</head>
<body style="font-family: Arial, sans-serif; background-color: #f9f9f9; padding: 20px;">
	<div style="max-width: 600px; margin: auto; background-color: white; padding: 20px; border-radius: 10px;">
		<h2 style="color: #333;">Confirmation de votre commande</h2>
		<p>Bonjour,</p>
		<p>Votre commande a été confirmée avec succès.</p>
		
		<h3>Détails de la commande</h3>
		<table style="width: 100%%; border-collapse: collapse; margin: 20px 0;">
			<thead>
				<tr style="background-color: #f0f0f0;">
					<th style="padding: 10px; text-align: left; border: 1px solid #ddd;">Produit</th>
					<th style="padding: 10px; text-align: left; border: 1px solid #ddd;">Quantité</th>
					<th style="padding: 10px; text-align: left; border: 1px solid #ddd;">Prix unitaire</th>
					<th style="padding: 10px; text-align: left; border: 1px solid #ddd;">Total</th>
				</tr>
			</thead>
			<tbody>
				%s
			</tbody>
			<tfoot>
				<tr>
					<td colspan="3" style="padding: 10px; text-align: right; font-weight: bold;">Total:</td>
					<td style="padding: 10px; font-weight: bold;">%.2f€</td>
				</tr>
			</tfoot>
		</table>
		
		<p style="margin-top: 30px; color: #555;">
			Cordialement,<br>
			<strong>L'équipe Cedra</strong>
		</p>
	</div>
</body>
</html>`, itemsHTML, order.TotalPrice)
}

// GenerateInvoicePDF génère un PDF de facture (utilise RenderReactInvoicePDF)
func GenerateInvoicePDF(order models.Order, userEmail string) ([]byte, error) {
	// Pour l'instant, on utilise RenderReactInvoicePDF avec l'ID de commande
	orderID := order.ID.String()
	frontURL := GetFrontendInvoiceBaseURL()
	
	// Générer le QR SEPA
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
	ref := fmt.Sprintf("FACT-%s", orderID)
	
	qrBase64, err := GenerateSepaQR(iban, bic, companyName, ref, order.TotalPrice)
	if err != nil {
		return nil, fmt.Errorf("erreur génération QR: %v", err)
	}
	
	return RenderReactInvoicePDF(frontURL, orderID, qrBase64)
}
