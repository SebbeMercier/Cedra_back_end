package user

import (
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/utils"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

// ================== CHANGE PASSWORD (avec ancien mot de passe) ==================

// POST /api/auth/change-password
func ChangePassword(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur non authentifié"})
		return
	}

	var input struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(input.NewPassword) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le nouveau mot de passe doit contenir au moins 8 caractères"})
		return
	}

	session, err := database.GetUsersSession()
	if err != nil {
		log.Printf("❌ Erreur session ScyllaDB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	id := fmt.Sprintf("%v", userID)
	uid, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID utilisateur invalide"})
		return
	}
	userUUID := gocql.UUID(uid)

	var password, provider string
	err = session.Query("SELECT password, provider FROM users WHERE user_id = ?", userUUID).Scan(&password, &provider)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur introuvable"})
		return
	}

	// Vérifie que c'est un compte local (pas OAuth)
	if provider != "local" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Les comptes OAuth ne peuvent pas changer de mot de passe ici"})
		return
	}

	// Vérifie l'ancien mot de passe
	valid, err := utils.VerifyPassword(input.OldPassword, password)
	if err != nil || !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Ancien mot de passe incorrect"})
		return
	}

	// Hash du nouveau mot de passe avec Argon2id
	hashedPassword, err := utils.HashPassword(input.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors du changement de mot de passe"})
		return
	}

	// Met à jour le mot de passe
	now := time.Now()
	err = session.Query("UPDATE users SET password = ?, updated_at = ? WHERE user_id = ?",
		hashedPassword, now, userUUID).Exec()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Mot de passe changé avec succès"})
}

// ================== FORGOT PASSWORD (demande de réinitialisation) ==================

// POST /api/auth/forgot-password
func ForgotPassword(c *gin.Context) {
	var input struct {
		Email string `json:"email"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := database.GetUsersSession()
	if err != nil {
		log.Printf("❌ Erreur session ScyllaDB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	// Récupérer user_id depuis users_by_email
	var userID gocql.UUID
	err = session.Query("SELECT user_id FROM users_by_email WHERE email = ?", input.Email).Scan(&userID)
	if err != nil {
		// ⚠️ Pour la sécurité, on ne révèle pas si l'email existe ou non
		c.JSON(http.StatusOK, gin.H{"message": "Si cet email existe, un lien de réinitialisation a été envoyé"})
		return
	}

	// Vérifier que c'est un compte local
	var provider string
	err = session.Query("SELECT provider FROM users WHERE user_id = ?", userID).Scan(&provider)
	if err != nil || provider != "local" {
		c.JSON(http.StatusOK, gin.H{"message": "Si cet email existe, un lien de réinitialisation a été envoyé"})
		return
	}

	// Récupérer le nom pour l'email
	var name string
	err = session.Query("SELECT name FROM users WHERE user_id = ?", userID).Scan(&name)
	if err != nil {
		name = ""
	}

	userIDStr := userID.String()

	// Génère un token de réinitialisation
	resetToken := generateResetToken()

	// Sauvegarde le token dans Redis (valide 1 heure)
	ctx := context.Background()
	err = database.RedisClient.Set(ctx, "reset_token:"+resetToken, userIDStr, 1*time.Hour).Err()
	if err != nil {
		log.Printf("❌ Erreur sauvegarde token reset: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la génération du lien"})
		return
	}

	// Envoie l'email
	go sendPasswordResetEmail(input.Email, name, resetToken)

	c.JSON(http.StatusOK, gin.H{"message": "Si cet email existe, un lien de réinitialisation a été envoyé"})
}

// POST /api/auth/reset-password
func ResetPassword(c *gin.Context) {
	var input struct {
		Token       string `json:"token"`
		NewPassword string `json:"newPassword"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(input.NewPassword) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le mot de passe doit contenir au moins 8 caractères"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Récupère l'user_id associé au token
	userID, err := database.RedisClient.Get(ctx, "reset_token:"+input.Token).Result()
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalide ou expiré"})
		return
	}

	// Hash du nouveau mot de passe avec Argon2id
	hashedPassword, err := utils.HashPassword(input.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la réinitialisation"})
		return
	}

	// Parse userID
	uid, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID utilisateur invalide"})
		return
	}
	userUUID := gocql.UUID(uid)

	// Met à jour le mot de passe
	session, err := database.GetUsersSession()
	if err != nil {
		log.Printf("❌ Erreur session ScyllaDB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	now := time.Now()
	err = session.Query("UPDATE users SET password = ?, updated_at = ? WHERE user_id = ?",
		hashedPassword, now, userUUID).Exec()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	// Supprime le token (usage unique)
	database.RedisClient.Del(ctx, "reset_token:"+input.Token)

	c.JSON(http.StatusOK, gin.H{"message": "Mot de passe réinitialisé avec succès"})
}

// ================== UTILITAIRES ==================

func generateResetToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func sendPasswordResetEmail(email, name, token string) {
	baseURL := os.Getenv("FRONTEND_URL")
	if baseURL == "" {
		baseURL = "https://cedra.eldocam.com"
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", baseURL, token)

	subject := "Réinitialisation de votre mot de passe Cedra"

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="fr">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
	<title>Réinitialisation de mot de passe</title>
</head>
<body style="font-family: Arial, sans-serif; background-color: #f9f9f9; padding: 20px;">
	<div style="max-width: 600px; margin: auto; background-color: white; padding: 20px; border-radius: 10px;">
		<h2 style="color: #333;">Réinitialisation de votre mot de passe</h2>
		<p>Bonjour <b>%s</b>,</p>
		<p>Vous avez demandé à réinitialiser votre mot de passe Cedra.</p>

		<p style="text-align: center; margin: 30px 0;">
			<a href="%s" style="background-color: #007bff; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Réinitialiser mon mot de passe</a>
		</p>

		<p style="font-size: 14px; color: #888; border-left: 3px solid #ffa500; padding-left: 15px; margin-top: 20px;">
			<strong>⚠️ Attention :</strong> Ce lien est valable pendant 1 heure seulement.
		</p>

		<p style="font-size: 14px; color: #888; margin-top: 20px;">
			Si vous n'avez pas demandé cette réinitialisation, ignorez simplement cet email. Votre mot de passe actuel restera inchangé.
		</p>

		<p style="margin-top: 30px; color: #555;">
			Cordialement,<br>
			<strong>L'équipe Cedra</strong>
		</p>
	</div>
</body>
</html>
	`, name, resetLink)

	err := utils.SendConfirmationEmail(email, subject, htmlBody, nil)

	if err != nil {
		log.Printf("❌ Erreur envoi email reset à %s: %v", email, err)
	} else {
		log.Printf("✅ Email de réinitialisation envoyé à %s", email)
	}
}
