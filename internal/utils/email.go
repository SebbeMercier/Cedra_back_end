package utils

import (
	"bytes"
	"os"
	"cedra_back_end/internal/models"
	"fmt"
	"github.com/wneessen/go-mail"
	"log"
	"strings"
	"github.com/jung-kurt/gofpdf"
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

func GenerateOrderConfirmationHTML(order models.Order, userEmail string) string {
	var itemsHTML strings.Builder
	for _, item := range order.Items {
		itemsHTML.WriteString(fmt.Sprintf(`
    	<tr>
			<td style="padding: 8px; border: 1px solid #eee;">%s</td>
			<td style="padding: 8px; border: 1px solid #eee;">%d</td>
			<td style="padding: 8px; border: 1px solid #eee;">%.2f €</td>
    	</tr>`, item.Name, item.Quantity, item.Price))
	}

	return fmt.Sprintf(`
		<!DOCTYPE html>
		<html lang="fr">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
			<title>Confirmation de commande</title>
		</head>
		<body style="font-family: Arial, sans-serif; background-color: #f9f9f9; padding: 20px;">
			<div style="max-width: 600px; margin: auto; background-color: white; padding: 20px; border-radius: 10px;">
				<h2 style="color: #333;">Merci pour votre commande !</h2>
				<p>Bonjour <b>%s</b>,</p>
				<p>Votre commande <b>%s</b> a bien été enregistrée.</p>

				<h3>Détails de la commande :</h3>
				<table style="width: 100%%; border-collapse: collapse; margin-top: 10px;">
					<thead>
						<tr style="background-color: #f0f0f0;">
							<th style="padding: 10px; border: 1px solid #eee; text-align: left;">Produit</th>
							<th style="padding: 10px; border: 1px solid #eee;">Quantité</th>
							<th style="padding: 10px; border: 1px solid #eee;">Prix</th>
						</tr>
					</thead>
					<tbody>
						%s
					</tbody>
				</table>

				<p style="margin-top: 20px;"><strong>Total :</strong> %.2f €</p>

				<p>Vous recevrez un second e-mail lors de l’expédition de votre commande.</p>
				<p style="font-size: 14px; color: #888;">Si vous avez des questions, contactez-nous à tout moment.</p>

				<p style="margin-top: 30px;">– L’équipe Cedra</p>
			</div>
		</body>
		</html>
	`, userEmail, order.ID.Hex(), itemsHTML.String(), order.TotalPrice)
}

func GenerateInvoicePDF(order models.Order, userEmail string) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "Facture Cedra")
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 12)
	pdf.Cell(40, 10, fmt.Sprintf("Client : %s", userEmail))
	pdf.Ln(8)
	pdf.Cell(40, 10, fmt.Sprintf("Commande : %s", order.ID.Hex()))
	pdf.Ln(12)

	pdf.Cell(40, 10, "Détails de la commande :")
	pdf.Ln(10)

	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(60, 8, "Produit", "1", 0, "", false, 0, "")
	pdf.CellFormat(30, 8, "Quantité", "1", 0, "", false, 0, "")
	pdf.CellFormat(30, 8, "Prix", "1", 1, "", false, 0, "")

	pdf.SetFont("Arial", "", 12)
	for _, item := range order.Items {
		pdf.CellFormat(60, 8, item.Name, "1", 0, "", false, 0, "")
		pdf.CellFormat(30, 8, fmt.Sprintf("%d", item.Quantity), "1", 0, "", false, 0, "")
		pdf.CellFormat(30, 8, fmt.Sprintf("%.2f €", item.Price), "1", 1, "", false, 0, "")
	}

	pdf.Ln(10)
	pdf.Cell(40, 10, fmt.Sprintf("Total : %.2f €", order.TotalPrice))

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
