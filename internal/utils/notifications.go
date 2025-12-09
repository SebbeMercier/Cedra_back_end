package utils

import (
	"cedra_back_end/internal/models"
	"fmt"
	"log"
)

// SendOrderStatusEmail envoie un email de notification de changement de statut
func SendOrderStatusEmail(order models.Order, userEmail string, newStatus string) error {
	subject := getStatusEmailSubject(newStatus)
	html := generateStatusEmailHTML(order, newStatus)

	err := SendConfirmationEmail(userEmail, subject, html, nil)
	if err != nil {
		log.Printf("❌ Erreur envoi email statut: %v", err)
		return err
	}

	log.Printf("📧 Email de statut envoyé: %s → %s", newStatus, userEmail)
	return nil
}

func getStatusEmailSubject(status string) string {
	switch status {
	case "paid":
		return "✅ Paiement confirmé - Cedra"
	case "shipped":
		return "📦 Votre commande a été expédiée - Cedra"
	case "delivered":
		return "🎉 Votre commande a été livrée - Cedra"
	case "cancelled":
		return "❌ Commande annulée - Cedra"
	case "refunded":
		return "💰 Remboursement effectué - Cedra"
	default:
		return "📋 Mise à jour de votre commande - Cedra"
	}
}

func generateStatusEmailHTML(order models.Order, status string) string {
	statusMessage := getStatusMessage(status)
	statusIcon := getStatusIcon(status)
	statusColor := getStatusColor(status)

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="fr">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Mise à jour de commande</title>
</head>
<body style="margin: 0; padding: 0; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; background-color: #f5f5f5;">
    <table role="presentation" style="width: 100%%; border-collapse: collapse; background-color: #f5f5f5;">
        <tr>
            <td style="padding: 40px 20px;">
                <table role="presentation" style="max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 12px; box-shadow: 0 4px 6px rgba(0,0,0,0.1);">
                    <!-- Header -->
                    <tr>
                        <td style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 40px 30px; text-align: center; border-radius: 12px 12px 0 0;">
                            <h1 style="margin: 0; color: #ffffff; font-size: 28px; font-weight: 600;">
                                %s Cedra
                            </h1>
                            <p style="margin: 10px 0 0 0; color: #ffffff; font-size: 16px; opacity: 0.9;">
                                Mise à jour de votre commande
                            </p>
                        </td>
                    </tr>
                    
                    <!-- Status Badge -->
                    <tr>
                        <td style="padding: 30px 30px 0 30px; text-align: center;">
                            <div style="display: inline-block; padding: 12px 24px; background-color: %s; color: #ffffff; border-radius: 25px; font-weight: 600; font-size: 14px; text-transform: uppercase; letter-spacing: 0.5px;">
                                %s %s
                            </div>
                        </td>
                    </tr>
                    
                    <!-- Content -->
                    <tr>
                        <td style="padding: 30px;">
                            <p style="margin: 0 0 20px 0; color: #333333; font-size: 16px; line-height: 1.6;">
                                %s
                            </p>
                            
                            <!-- Order Info Box -->
                            <table role="presentation" style="width: 100%%; border-collapse: collapse; background-color: #f8f9fa; border-radius: 8px; margin: 20px 0;">
                                <tr>
                                    <td style="padding: 20px;">
                                        <h3 style="margin: 0 0 15px 0; color: #333333; font-size: 18px; font-weight: 600;">
                                            📦 Détails de la commande
                                        </h3>
                                        <table role="presentation" style="width: 100%%; border-collapse: collapse;">
                                            <tr>
                                                <td style="padding: 8px 0; color: #666666; font-size: 14px;">
                                                    <strong style="color: #333333;">Numéro de commande:</strong>
                                                </td>
                                                <td style="padding: 8px 0; color: #333333; font-size: 14px; text-align: right;">
                                                    #%s
                                                </td>
                                            </tr>
                                            <tr>
                                                <td style="padding: 8px 0; color: #666666; font-size: 14px;">
                                                    <strong style="color: #333333;">Montant total:</strong>
                                                </td>
                                                <td style="padding: 8px 0; color: #333333; font-size: 14px; text-align: right; font-weight: 600;">
                                                    %.2f€
                                                </td>
                                            </tr>
                                            <tr>
                                                <td style="padding: 8px 0; color: #666666; font-size: 14px;">
                                                    <strong style="color: #333333;">Statut:</strong>
                                                </td>
                                                <td style="padding: 8px 0; color: %s; font-size: 14px; text-align: right; font-weight: 600;">
                                                    %s
                                                </td>
                                            </tr>
                                        </table>
                                    </td>
                                </tr>
                            </table>
                            
                            <!-- CTA Button -->
                            <table role="presentation" style="width: 100%%; margin: 30px 0;">
                                <tr>
                                    <td style="text-align: center;">
                                        <a href="http://cedra.eldocam.com:5173/orders" style="display: inline-block; padding: 14px 32px; background-color: #667eea; color: #ffffff; text-decoration: none; border-radius: 6px; font-weight: 600; font-size: 15px;">
                                            Voir ma commande
                                        </a>
                                    </td>
                                </tr>
                            </table>
                            
                            <p style="margin: 20px 0 0 0; color: #666666; font-size: 14px; line-height: 1.6;">
                                Vous avez des questions ? Notre équipe support est disponible 7j/7 pour vous aider.
                            </p>
                        </td>
                    </tr>
                    
                    <!-- Footer -->
                    <tr>
                        <td style="padding: 30px; background-color: #f8f9fa; border-radius: 0 0 12px 12px; text-align: center;">
                            <p style="margin: 0 0 10px 0; color: #999999; font-size: 12px;">
                                © 2024 Cedra - Tous droits réservés
                            </p>
                            <p style="margin: 0; color: #999999; font-size: 12px;">
                                Cet email a été envoyé automatiquement, merci de ne pas y répondre.
                            </p>
                            <p style="margin: 10px 0 0 0; color: #999999; font-size: 12px;">
                                <a href="http://cedra.eldocam.com:5173" style="color: #667eea; text-decoration: none;">Visiter notre site</a> • 
                                <a href="http://cedra.eldocam.com:5173/support" style="color: #667eea; text-decoration: none;">Support</a>
                            </p>
                        </td>
                    </tr>
                </table>
            </td>
        </tr>
    </table>
</body>
</html>
`, statusIcon, statusColor, statusIcon, status, statusMessage, order.ID.String()[:8], order.TotalPrice, statusColor, status)

	return html
}

func getStatusMessage(status string) string {
	switch status {
	case "paid":
		return "Votre paiement a été confirmé avec succès. Nous préparons votre commande."
	case "shipped":
		return "Bonne nouvelle ! Votre commande a été expédiée et est en route vers vous."
	case "delivered":
		return "Votre commande a été livrée avec succès. Nous espérons que vous en êtes satisfait !"
	case "cancelled":
		return "Votre commande a été annulée. Si vous avez des questions, n'hésitez pas à nous contacter."
	case "refunded":
		return "Votre remboursement a été traité. Les fonds seront crédités sur votre compte sous 5-10 jours ouvrés."
	default:
		return "Le statut de votre commande a été mis à jour."
	}
}

func getStatusIcon(status string) string {
	switch status {
	case "paid":
		return "✅"
	case "shipped":
		return "📦"
	case "delivered":
		return "🎉"
	case "cancelled":
		return "❌"
	case "refunded":
		return "💰"
	default:
		return "📋"
	}
}

func getStatusColor(status string) string {
	switch status {
	case "paid":
		return "#10b981" // Green
	case "shipped":
		return "#3b82f6" // Blue
	case "delivered":
		return "#8b5cf6" // Purple
	case "cancelled":
		return "#ef4444" // Red
	case "refunded":
		return "#f59e0b" // Orange
	default:
		return "#6b7280" // Gray
	}
}

// SendRefundRequestEmail envoie un email de confirmation de demande de remboursement
func SendRefundRequestEmail(userEmail string, orderID string, reason string) error {
	subject := "💰 Demande de remboursement reçue - Cedra"

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 30px; text-align: center; border-radius: 10px 10px 0 0; }
        .content { background: #f9f9f9; padding: 30px; border-radius: 0 0 10px 10px; }
        .info-box { background: white; padding: 20px; border-radius: 8px; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>💰 Demande de remboursement</h1>
        </div>
        <div class="content">
            <p>Nous avons bien reçu votre demande de remboursement.</p>
            
            <div class="info-box">
                <h3>Détails</h3>
                <p><strong>Commande:</strong> %s</p>
                <p><strong>Raison:</strong> %s</p>
                <p><strong>Statut:</strong> En attente de traitement</p>
            </div>

            <p>Notre équipe va examiner votre demande dans les plus brefs délais. Vous recevrez une notification par email une fois la décision prise.</p>
            
            <p>Délai de traitement habituel : 2-5 jours ouvrés</p>
        </div>
    </div>
</body>
</html>
`, orderID, reason)

	return SendConfirmationEmail(userEmail, subject, html, nil)
}

// SendWelcomeEmail envoie un email de bienvenue
func SendWelcomeEmail(userEmail, userName string) error {
	subject := "🎉 Bienvenue sur Cedra !"

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 30px; text-align: center; border-radius: 10px 10px 0 0; }
        .content { background: #f9f9f9; padding: 30px; border-radius: 0 0 10px 10px; }
        .cta-button { display: inline-block; padding: 15px 30px; background: #667eea; color: white; text-decoration: none; border-radius: 5px; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎉 Bienvenue %s !</h1>
        </div>
        <div class="content">
            <p>Merci de vous être inscrit sur Cedra, votre nouvelle destination shopping en ligne.</p>
            
            <p>Découvrez dès maintenant notre sélection de produits et profitez de nos offres exclusives !</p>
            
            <a href="#" class="cta-button">Commencer mes achats</a>
            
            <h3>Avantages membres :</h3>
            <ul>
                <li>✅ Livraison gratuite dès 50€</li>
                <li>✅ Retours gratuits sous 30 jours</li>
                <li>✅ Codes promo exclusifs</li>
                <li>✅ Support client 7j/7</li>
            </ul>
        </div>
    </div>
</body>
</html>
`, userName)

	return SendConfirmationEmail(userEmail, subject, html, nil)
}

// SendRefundApprovedEmail envoie un email de remboursement approuvé
func SendRefundApprovedEmail(userEmail string, orderID string, amount float64) error {
	subject := "✅ Remboursement approuvé - Cedra"

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="fr">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Remboursement approuvé</title>
</head>
<body style="margin: 0; padding: 0; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; background-color: #f5f5f5;">
    <table role="presentation" style="width: 100%%; border-collapse: collapse; background-color: #f5f5f5;">
        <tr>
            <td style="padding: 40px 20px;">
                <table role="presentation" style="max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 12px; box-shadow: 0 4px 6px rgba(0,0,0,0.1);">
                    <!-- Header -->
                    <tr>
                        <td style="background: linear-gradient(135deg, #10b981 0%%, #059669 100%%); padding: 40px 30px; text-align: center; border-radius: 12px 12px 0 0;">
                            <h1 style="margin: 0; color: #ffffff; font-size: 28px; font-weight: 600;">
                                ✅ Remboursement approuvé
                            </h1>
                            <p style="margin: 10px 0 0 0; color: #ffffff; font-size: 16px; opacity: 0.9;">
                                Votre demande a été acceptée
                            </p>
                        </td>
                    </tr>
                    
                    <!-- Content -->
                    <tr>
                        <td style="padding: 40px 30px;">
                            <p style="margin: 0 0 25px 0; color: #333333; font-size: 16px; line-height: 1.6;">
                                Bonne nouvelle ! Votre demande de remboursement a été approuvée.
                            </p>
                            
                            <!-- Amount Box -->
                            <table role="presentation" style="width: 100%%; border-collapse: collapse; background: linear-gradient(135deg, #d1fae5 0%%, #a7f3d0 100%%); border-radius: 12px; margin: 25px 0;">
                                <tr>
                                    <td style="padding: 30px; text-align: center;">
                                        <p style="margin: 0 0 10px 0; color: #065f46; font-size: 14px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.5px;">
                                            Montant remboursé
                                        </p>
                                        <p style="margin: 0; color: #047857; font-size: 42px; font-weight: 700;">
                                            %.2f€
                                        </p>
                                    </td>
                                </tr>
                            </table>
                            
                            <!-- Info Box -->
                            <table role="presentation" style="width: 100%%; border-collapse: collapse; background-color: #f8f9fa; border-radius: 8px; margin: 25px 0;">
                                <tr>
                                    <td style="padding: 20px;">
                                        <h3 style="margin: 0 0 15px 0; color: #333333; font-size: 18px; font-weight: 600;">
                                            📋 Informations
                                        </h3>
                                        <table role="presentation" style="width: 100%%; border-collapse: collapse;">
                                            <tr>
                                                <td style="padding: 8px 0; color: #666666; font-size: 14px;">
                                                    <strong>Commande:</strong>
                                                </td>
                                                <td style="padding: 8px 0; color: #333333; font-size: 14px; text-align: right;">
                                                    #%s
                                                </td>
                                            </tr>
                                            <tr>
                                                <td style="padding: 8px 0; color: #666666; font-size: 14px;">
                                                    <strong>Délai de traitement:</strong>
                                                </td>
                                                <td style="padding: 8px 0; color: #333333; font-size: 14px; text-align: right;">
                                                    5-10 jours ouvrés
                                                </td>
                                            </tr>
                                        </table>
                                    </td>
                                </tr>
                            </table>
                            
                            <table role="presentation" style="width: 100%%; border-collapse: collapse; background-color: #eff6ff; border-left: 4px solid #3b82f6; border-radius: 8px; margin: 25px 0;">
                                <tr>
                                    <td style="padding: 20px;">
                                        <p style="margin: 0; color: #1e40af; font-size: 14px; line-height: 1.6;">
                                            <strong>ℹ️ Important:</strong> Le remboursement sera crédité sur le moyen de paiement utilisé lors de l'achat. Selon votre banque, le délai peut varier.
                                        </p>
                                    </td>
                                </tr>
                            </table>
                            
                            <!-- CTA Button -->
                            <table role="presentation" style="width: 100%%; margin: 30px 0;">
                                <tr>
                                    <td style="text-align: center;">
                                        <a href="http://cedra.eldocam.com:5173/refunds" style="display: inline-block; padding: 14px 32px; background-color: #10b981; color: #ffffff; text-decoration: none; border-radius: 6px; font-weight: 600; font-size: 15px;">
                                            Voir mes remboursements
                                        </a>
                                    </td>
                                </tr>
                            </table>
                        </td>
                    </tr>
                    
                    <!-- Footer -->
                    <tr>
                        <td style="padding: 30px; background-color: #f8f9fa; border-radius: 0 0 12px 12px; text-align: center;">
                            <p style="margin: 0 0 10px 0; color: #999999; font-size: 12px;">
                                © 2024 Cedra - Tous droits réservés
                            </p>
                            <p style="margin: 0; color: #999999; font-size: 12px;">
                                Merci de votre confiance
                            </p>
                        </td>
                    </tr>
                </table>
            </td>
        </tr>
    </table>
</body>
</html>
`, amount, orderID[:8])

	return SendConfirmationEmail(userEmail, subject, html, nil)
}

// SendRefundRejectedEmail envoie un email de remboursement rejeté
func SendRefundRejectedEmail(userEmail string, orderID string, reason string) error {
	subject := "❌ Demande de remboursement refusée - Cedra"

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="fr">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Remboursement refusé</title>
</head>
<body style="margin: 0; padding: 0; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; background-color: #f5f5f5;">
    <table role="presentation" style="width: 100%%; border-collapse: collapse; background-color: #f5f5f5;">
        <tr>
            <td style="padding: 40px 20px;">
                <table role="presentation" style="max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 12px; box-shadow: 0 4px 6px rgba(0,0,0,0.1);">
                    <!-- Header -->
                    <tr>
                        <td style="background: linear-gradient(135deg, #ef4444 0%%, #dc2626 100%%); padding: 40px 30px; text-align: center; border-radius: 12px 12px 0 0;">
                            <h1 style="margin: 0; color: #ffffff; font-size: 28px; font-weight: 600;">
                                Demande de remboursement
                            </h1>
                            <p style="margin: 10px 0 0 0; color: #ffffff; font-size: 16px; opacity: 0.9;">
                                Mise à jour de votre demande
                            </p>
                        </td>
                    </tr>
                    
                    <!-- Content -->
                    <tr>
                        <td style="padding: 40px 30px;">
                            <p style="margin: 0 0 25px 0; color: #333333; font-size: 16px; line-height: 1.6;">
                                Après examen de votre demande, nous ne pouvons malheureusement pas procéder au remboursement.
                            </p>
                            
                            <!-- Info Box -->
                            <table role="presentation" style="width: 100%%; border-collapse: collapse; background-color: #fef2f2; border-left: 4px solid #ef4444; border-radius: 8px; margin: 25px 0;">
                                <tr>
                                    <td style="padding: 20px;">
                                        <h3 style="margin: 0 0 15px 0; color: #991b1b; font-size: 18px; font-weight: 600;">
                                            📋 Raison du refus
                                        </h3>
                                        <p style="margin: 0; color: #7f1d1d; font-size: 14px; line-height: 1.6;">
                                            %s
                                        </p>
                                    </td>
                                </tr>
                            </table>
                            
                            <table role="presentation" style="width: 100%%; border-collapse: collapse; background-color: #f8f9fa; border-radius: 8px; margin: 25px 0;">
                                <tr>
                                    <td style="padding: 20px;">
                                        <p style="margin: 0 0 10px 0; color: #666666; font-size: 14px;">
                                            <strong>Commande:</strong> #%s
                                        </p>
                                        <p style="margin: 0; color: #666666; font-size: 14px;">
                                            Si vous pensez qu'il s'agit d'une erreur ou si vous avez des questions, notre équipe support est à votre disposition.
                                        </p>
                                    </td>
                                </tr>
                            </table>
                            
                            <!-- CTA Button -->
                            <table role="presentation" style="width: 100%%; margin: 30px 0;">
                                <tr>
                                    <td style="text-align: center;">
                                        <a href="http://cedra.eldocam.com:5173/support" style="display: inline-block; padding: 14px 32px; background-color: #667eea; color: #ffffff; text-decoration: none; border-radius: 6px; font-weight: 600; font-size: 15px;">
                                            Contacter le support
                                        </a>
                                    </td>
                                </tr>
                            </table>
                        </td>
                    </tr>
                    
                    <!-- Footer -->
                    <tr>
                        <td style="padding: 30px; background-color: #f8f9fa; border-radius: 0 0 12px 12px; text-align: center;">
                            <p style="margin: 0 0 10px 0; color: #999999; font-size: 12px;">
                                © 2024 Cedra - Tous droits réservés
                            </p>
                            <p style="margin: 0; color: #999999; font-size: 12px;">
                                Support: support@cedra.com
                            </p>
                        </td>
                    </tr>
                </table>
            </td>
        </tr>
    </table>
</body>
</html>
`, reason, orderID[:8])

	return SendConfirmationEmail(userEmail, subject, html, nil)
}
