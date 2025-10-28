package user

import (
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
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
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	id := fmt.Sprintf("%v", userID)

	var user models.User
	col := database.MongoAuthDB.Collection("users")

	err := col.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur introuvable"})
		return
	}

	// Vérifie que c'est un compte local (pas OAuth)
	if user.Provider != "local" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Les comptes OAuth ne peuvent pas changer de mot de passe ici"})
		return
	}

	// Vérifie l'ancien mot de passe
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.OldPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Ancien mot de passe incorrect"})
		return
	}

	// Hash du nouveau mot de passe
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors du changement de mot de passe"})
		return
	}

	// Met à jour le mot de passe
	_, err = col.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{"password": string(hashedPassword)},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	log.Printf("✅ Mot de passe changé pour l'utilisateur %s", user.Email)

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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	col := database.MongoAuthDB.Collection("users")

	err := col.FindOne(ctx, bson.M{"email": input.Email, "provider": "local"}).Decode(&user)
	if err != nil {
		// ⚠️ Pour la sécurité, on ne révèle pas si l'email existe ou non
		c.JSON(http.StatusOK, gin.H{"message": "Si cet email existe, un lien de réinitialisation a été envoyé"})
		return
	}

	// Génère un token de réinitialisation
	resetToken := generateResetToken()
	resetExpiry := time.Now().Add(1 * time.Hour) // Valide 1 heure

	// Sauvegarde le token dans Redis (ou MongoDB si vous préférez)
	err = database.RedisClient.Set(ctx, "reset_token:"+resetToken, user.ID, 1*time.Hour).Err()
	if err != nil {
		log.Printf("❌ Erreur sauvegarde token reset: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la génération du lien"})
		return
	}

	// Envoie l'email
	go sendPasswordResetEmail(user.Email, user.Name, resetToken)

	log.Printf("✅ Email de réinitialisation envoyé à %s (token valide jusqu'à %s)", user.Email, resetExpiry.Format("15:04:05"))

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

	// Hash du nouveau mot de passe
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la réinitialisation"})
		return
	}

	// Met à jour le mot de passe
	col := database.MongoAuthDB.Collection("users")
	result, err := col.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{
		"$set": bson.M{"password": string(hashedPassword)},
	})

	if err != nil || result.MatchedCount == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	// Supprime le token (usage unique)
	database.RedisClient.Del(ctx, "reset_token:"+input.Token)

	log.Printf("✅ Mot de passe réinitialisé pour l'utilisateur %s", userID)

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
