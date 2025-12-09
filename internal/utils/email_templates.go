package utils

import "fmt"

// SendWelcomeEmail envoie un email de bienvenue moderne
func SendWelcomeEmail(userEmail, userName string) error {
	subject := "🎉 Bienvenue sur Cedra !"

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="fr">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Bienvenue sur Cedra</title>
</head>
<body style="margin: 0; padding: 0; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; background-color: #f5f5f5;">
    <table role="presentation" style="width: 100%%; border-collapse: collapse; background-color: #f5f5f5;">
        <tr>
            <td style="padding: 40px 20px;">
                <table role="presentation" style="max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 12px; box-shadow: 0 4px 6px rgba(0,0,0,0.1);">
                    <!-- Header -->
                    <tr>
                        <td style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 50px 30px; text-align: center; border-radius: 12px 12px 0 0;">
                            <h1 style="margin: 0; color: #ffffff; font-size: 32px; font-weight: 700;">
                                🎉 Bienvenue sur Cedra !
                            </h1>
                            <p style="margin: 15px 0 0 0; color: #ffffff; font-size: 18px; opacity: 0.95;">
                                Bonjour %s
                            </p>
                        </td>
                    </tr>
                    
                    <!-- Content -->
                    <tr>
                        <td style="padding: 40px 30px;">
                            <p style="margin: 0 0 25px 0; color: #333333; font-size: 16px; line-height: 1.6;">
                                Merci de vous être inscrit sur <strong>Cedra</strong>, votre nouvelle destination shopping en ligne ! 🛍️
                            </p>
                            
                            <p style="margin: 0 0 30px 0; color: #333333; font-size: 16px; line-height: 1.6;">
                                Découvrez dès maintenant notre sélection de produits et profitez de nos offres exclusives.
                            </p>
                            
                            <!-- CTA Button -->
                            <table role="presentation" style="width: 100%%; margin: 30px 0;">
                                <tr>
                                    <td style="text-align: center;">
                                        <a href="http://cedra.eldocam.com:5173/products" style="display: inline-block; padding: 16px 40px; background-color: #667eea; color: #ffffff; text-decoration: none; border-radius: 8px; font-weight: 600; font-size: 16px; box-shadow: 0 4px 6px rgba(102, 126, 234, 0.3);">
                                            🛍️ Commencer mes achats
                                        </a>
                                    </td>
                                </tr>
                            </table>
                            
                            <!-- Benefits -->
                            <table role="presentation" style="width: 100%%; border-collapse: collapse; margin: 30px 0;">
                                <tr>
                                    <td style="padding: 25px; background-color: #f8f9fa; border-radius: 8px;">
                                        <h3 style="margin: 0 0 20px 0; color: #333333; font-size: 20px; font-weight: 600; text-align: center;">
                                            🎁 Vos avantages membres
                                        </h3>
                                        <table role="presentation" style="width: 100%%; border-collapse: collapse;">
                                            <tr>
                                                <td style="padding: 12px 0;">
                                                    <table role="presentation" style="width: 100%%; border-collapse: collapse;">
                                                        <tr>
                                                            <td style="width: 40px; vertical-align: top;">
                                                                <div style="width: 32px; height: 32px; background-color: #10b981; border-radius: 50%%; display: flex; align-items: center; justify-content: center; color: #ffffff; font-size: 18px;">
                                                                    ✓
                                                                </div>
                                                            </td>
                                                            <td style="padding-left: 15px;">
                                                                <p style="margin: 0; color: #333333; font-size: 15px; font-weight: 600;">
                                                                    Livraison gratuite dès 50€
                                                                </p>
                                                                <p style="margin: 5px 0 0 0; color: #666666; font-size: 13px;">
                                                                    Profitez de la livraison offerte sur toutes vos commandes
                                                                </p>
                                                            </td>
                                                        </tr>
                                                    </table>
                                                </td>
                                            </tr>
                                            <tr>
                                                <td style="padding: 12px 0;">
                                                    <table role="presentation" style="width: 100%%; border-collapse: collapse;">
                                                        <tr>
                                                            <td style="width: 40px; vertical-align: top;">
                                                                <div style="width: 32px; height: 32px; background-color: #3b82f6; border-radius: 50%%; display: flex; align-items: center; justify-content: center; color: #ffffff; font-size: 18px;">
                                                                    ✓
                                                                </div>
                                                            </td>
                                                            <td style="padding-left: 15px;">
                                                                <p style="margin: 0; color: #333333; font-size: 15px; font-weight: 600;">
                                                                    Retours gratuits sous 30 jours
                                                                </p>
                                                                <p style="margin: 5px 0 0 0; color: #666666; font-size: 13px;">
                                                                    Changez d'avis ? Retour simple et gratuit
                                                                </p>
                                                            </td>
                                                        </tr>
                                                    </table>
                                                </td>
                                            </tr>
                                            <tr>
                                                <td style="padding: 12px 0;">
                                                    <table role="presentation" style="width: 100%%; border-collapse: collapse;">
                                                        <tr>
                                                            <td style="width: 40px; vertical-align: top;">
                                                                <div style="width: 32px; height: 32px; background-color: #f59e0b; border-radius: 50%%; display: flex; align-items: center; justify-content: center; color: #ffffff; font-size: 18px;">
                                                                    ✓
                                                                </div>
                                                            </td>
                                                            <td style="padding-left: 15px;">
                                                                <p style="margin: 0; color: #333333; font-size: 15px; font-weight: 600;">
                                                                    Codes promo exclusifs
                                                                </p>
                                                                <p style="margin: 5px 0 0 0; color: #666666; font-size: 13px;">
                                                                    Recevez des offres spéciales réservées aux membres
                                                                </p>
                                                            </td>
                                                        </tr>
                                                    </table>
                                                </td>
                                            </tr>
                                            <tr>
                                                <td style="padding: 12px 0;">
                                                    <table role="presentation" style="width: 100%%; border-collapse: collapse;">
                                                        <tr>
                                                            <td style="width: 40px; vertical-align: top;">
                                                                <div style="width: 32px; height: 32px; background-color: #8b5cf6; border-radius: 50%%; display: flex; align-items: center; justify-content: center; color: #ffffff; font-size: 18px;">
                                                                    ✓
                                                                </div>
                                                            </td>
                                                            <td style="padding-left: 15px;">
                                                                <p style="margin: 0; color: #333333; font-size: 15px; font-weight: 600;">
                                                                    Support client 7j/7
                                                                </p>
                                                                <p style="margin: 5px 0 0 0; color: #666666; font-size: 13px;">
                                                                    Notre équipe est là pour vous aider à tout moment
                                                                </p>
                                                            </td>
                                                        </tr>
                                                    </table>
                                                </td>
                                            </tr>
                                        </table>
                                    </td>
                                </tr>
                            </table>
                            
                            <!-- Promo Code -->
                            <table role="presentation" style="width: 100%%; border-collapse: collapse; background: linear-gradient(135deg, #fef3c7 0%%, #fde68a 100%%); border-radius: 8px; margin: 30px 0;">
                                <tr>
                                    <td style="padding: 25px; text-align: center;">
                                        <p style="margin: 0 0 10px 0; color: #92400e; font-size: 14px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.5px;">
                                            🎁 Cadeau de bienvenue
                                        </p>
                                        <p style="margin: 0 0 15px 0; color: #78350f; font-size: 16px;">
                                            Profitez de <strong>10%% de réduction</strong> sur votre première commande avec le code :
                                        </p>
                                        <div style="display: inline-block; padding: 12px 24px; background-color: #ffffff; border: 2px dashed #f59e0b; border-radius: 6px;">
                                            <p style="margin: 0; color: #92400e; font-size: 24px; font-weight: 700; letter-spacing: 2px;">
                                                WELCOME10
                                            </p>
                                        </div>
                                    </td>
                                </tr>
                            </table>
                        </td>
                    </tr>
                    
                    <!-- Footer -->
                    <tr>
                        <td style="padding: 30px; background-color: #f8f9fa; border-radius: 0 0 12px 12px; text-align: center;">
                            <p style="margin: 0 0 15px 0; color: #333333; font-size: 14px;">
                                Suivez-nous sur les réseaux sociaux
                            </p>
                            <p style="margin: 0 0 15px 0;">
                                <a href="#" style="display: inline-block; margin: 0 10px; color: #667eea; text-decoration: none; font-size: 24px;">📘</a>
                                <a href="#" style="display: inline-block; margin: 0 10px; color: #667eea; text-decoration: none; font-size: 24px;">📷</a>
                                <a href="#" style="display: inline-block; margin: 0 10px; color: #667eea; text-decoration: none; font-size: 24px;">🐦</a>
                            </p>
                            <p style="margin: 0 0 10px 0; color: #999999; font-size: 12px;">
                                © 2024 Cedra - Tous droits réservés
                            </p>
                            <p style="margin: 0; color: #999999; font-size: 12px;">
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
`, userName)

	return SendConfirmationEmail(userEmail, subject, html, nil)
}

// SendOrderConfirmationEmail envoie un email de confirmation de commande
func SendOrderConfirmationEmail(userEmail string, orderID string, totalAmount float64, items string) error {
	subject := "✅ Commande confirmée - Cedra"

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="fr">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Commande confirmée</title>
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
                                ✅ Commande confirmée !
                            </h1>
                            <p style="margin: 10px 0 0 0; color: #ffffff; font-size: 16px; opacity: 0.9;">
                                Merci pour votre achat
                            </p>
                        </td>
                    </tr>
                    
                    <!-- Content -->
                    <tr>
                        <td style="padding: 40px 30px;">
                            <p style="margin: 0 0 25px 0; color: #333333; font-size: 16px; line-height: 1.6;">
                                Votre commande a été confirmée avec succès ! Nous préparons votre colis avec soin.
                            </p>
                            
                            <!-- Order Summary -->
                            <table role="presentation" style="width: 100%%; border-collapse: collapse; background-color: #f8f9fa; border-radius: 8px; margin: 25px 0;">
                                <tr>
                                    <td style="padding: 20px;">
                                        <h3 style="margin: 0 0 15px 0; color: #333333; font-size: 18px; font-weight: 600;">
                                            📦 Récapitulatif de commande
                                        </h3>
                                        <table role="presentation" style="width: 100%%; border-collapse: collapse;">
                                            <tr>
                                                <td style="padding: 8px 0; color: #666666; font-size: 14px;">
                                                    <strong>Numéro de commande:</strong>
                                                </td>
                                                <td style="padding: 8px 0; color: #333333; font-size: 14px; text-align: right;">
                                                    #%s
                                                </td>
                                            </tr>
                                            <tr>
                                                <td style="padding: 8px 0; color: #666666; font-size: 14px;">
                                                    <strong>Montant total:</strong>
                                                </td>
                                                <td style="padding: 8px 0; color: #10b981; font-size: 18px; text-align: right; font-weight: 700;">
                                                    %.2f€
                                                </td>
                                            </tr>
                                        </table>
                                    </td>
                                </tr>
                            </table>
                            
                            <!-- Next Steps -->
                            <table role="presentation" style="width: 100%%; border-collapse: collapse; background-color: #eff6ff; border-left: 4px solid #3b82f6; border-radius: 8px; margin: 25px 0;">
                                <tr>
                                    <td style="padding: 20px;">
                                        <h4 style="margin: 0 0 10px 0; color: #1e40af; font-size: 16px; font-weight: 600;">
                                            📋 Prochaines étapes
                                        </h4>
                                        <ol style="margin: 0; padding-left: 20px; color: #1e3a8a; font-size: 14px; line-height: 1.8;">
                                            <li>Préparation de votre commande (1-2 jours)</li>
                                            <li>Expédition et envoi du numéro de suivi</li>
                                            <li>Livraison à votre adresse</li>
                                        </ol>
                                    </td>
                                </tr>
                            </table>
                            
                            <!-- CTA Button -->
                            <table role="presentation" style="width: 100%%; margin: 30px 0;">
                                <tr>
                                    <td style="text-align: center;">
                                        <a href="http://cedra.eldocam.com:5173/orders/%s" style="display: inline-block; padding: 14px 32px; background-color: #667eea; color: #ffffff; text-decoration: none; border-radius: 6px; font-weight: 600; font-size: 15px;">
                                            Suivre ma commande
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
                                Des questions ? Contactez-nous à support@cedra.com
                            </p>
                        </td>
                    </tr>
                </table>
            </td>
        </tr>
    </table>
</body>
</html>
`, orderID[:8], totalAmount, orderID[:8])

	return SendConfirmationEmail(userEmail, subject, html, nil)
}
