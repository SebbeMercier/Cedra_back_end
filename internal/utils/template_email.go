package utils

import (
	"bytes"
	"html/template"
	"log"
)

// SendWelcomeEmailFromTemplate envoie un email de bienvenue depuis le template React compilé
func SendWelcomeEmailFromTemplate(userEmail, userName string) error {
	// Charger le template HTML compilé depuis React
	tmpl, err := template.ParseFiles("internal/templates/welcome.html")
	if err != nil {
		log.Printf("❌ Erreur chargement template: %v", err)
		return err
	}

	// Préparer les données
	data := map[string]string{
		"UserName": userName,
	}

	// Exécuter le template avec les données
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		log.Printf("❌ Erreur exécution template: %v", err)
		return err
	}

	// Envoyer l'email
	subject := "🎉 Bienvenue sur Cedra !"
	err = SendConfirmationEmail(userEmail, subject, buf.String(), nil)
	if err != nil {
		log.Printf("❌ Erreur envoi email: %v", err)
		return err
	}

	log.Printf("📧 Email de bienvenue envoyé: %s", userEmail)
	return nil
}

// SendOrderConfirmationFromTemplate envoie un email de confirmation de commande depuis le template React
func SendOrderConfirmationFromTemplate(userEmail string, orderID string, totalAmount float64) error {
	// Charger le template HTML compilé depuis React
	tmpl, err := template.ParseFiles("internal/templates/order-confirmation.html")
	if err != nil {
		log.Printf("❌ Erreur chargement template: %v", err)
		return err
	}

	// Préparer les données
	data := map[string]interface{}{
		"OrderID":     orderID,
		"TotalAmount": totalAmount,
	}

	// Exécuter le template avec les données
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		log.Printf("❌ Erreur exécution template: %v", err)
		return err
	}

	// Envoyer l'email
	subject := "✅ Commande confirmée - Cedra"
	err = SendConfirmationEmail(userEmail, subject, buf.String(), nil)
	if err != nil {
		log.Printf("❌ Erreur envoi email: %v", err)
		return err
	}

	log.Printf("📧 Email de confirmation envoyé: %s (commande: %s)", userEmail, orderID)
	return nil
}
