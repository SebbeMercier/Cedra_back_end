package company

import (
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
	"cedra_back_end/internal/utils"
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

func GetMyCompany(c *gin.Context) {
	userID, exists := c.Get("user_id")
	log.Printf("🔍 DEBUG /company/me → user_id=%v (exists=%v)", userID, exists)

	if !exists || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non autorisé"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 🔹 Récupérer l'utilisateur
	var user models.User
	err := database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"_id": userID.(string),
	}).Decode(&user)

	if err != nil {
		log.Printf("❌ Utilisateur introuvable: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur introuvable"})
		return
	}

	if user.CompanyID == nil || *user.CompanyID == "" {
		log.Printf("⚠️ Aucune société associée pour l'utilisateur %s", user.Email)
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune société associée"})
		return
	}

	log.Printf("🔍 CompanyID de l'utilisateur: %s", *user.CompanyID)

	// ✅ Convertir le CompanyID string en ObjectID
	companyOID, err := primitive.ObjectIDFromHex(*user.CompanyID)
	if err != nil {
		log.Printf("❌ Erreur conversion CompanyID en ObjectID: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ID de société invalide"})
		return
	}

	// 🔹 Récupérer la société
	var company bson.M
	err = database.MongoCompanyDB.Collection("companies").FindOne(ctx, bson.M{
		"_id": companyOID, // ✅ Utiliser l'ObjectID au lieu du string
	}).Decode(&company)

	if err != nil {
		log.Printf("❌ Société non trouvée: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Société introuvable"})
		return
	}

	log.Printf("✅ Société trouvée: %v", company["name"])

	c.JSON(http.StatusOK, company)
}

func UpdateCompanyBilling(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var input struct {
		BillingStreet     string `json:"billingStreet"`
		BillingPostalCode string `json:"billingPostalCode"`
		BillingCity       string `json:"billingCity"`
		BillingCountry    string `json:"billingCountry"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err := database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"_id": userID.(string),
	}).Decode(&user)
	if err != nil || user.CompanyID == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune société associée"})
		return
	}

	// ✅ Convertir en ObjectID
	companyOID, err := primitive.ObjectIDFromHex(*user.CompanyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ID de société invalide"})
		return
	}

	update := bson.M{
		"$set": bson.M{
			"billingStreet":     input.BillingStreet,
			"billingPostalCode": input.BillingPostalCode,
			"billingCity":       input.BillingCity,
			"billingCountry":    input.BillingCountry,
		},
	}
	_, err = database.MongoCompanyDB.Collection("companies").UpdateOne(
		ctx,
		bson.M{"_id": companyOID}, // ✅ ObjectID
		update,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Adresse de facturation mise à jour"})
}

func ListCompanyEmployees(c *gin.Context) {
	userID, _ := c.Get("user_id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err := database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"_id": userID.(string),
	}).Decode(&user)
	if err != nil || user.CompanyID == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune société associée"})
		return
	}

	cursor, err := database.MongoAuthDB.Collection("users").Find(ctx, bson.M{
		"companyId": *user.CompanyID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération"})
		return
	}
	defer cursor.Close(ctx)

	var employees []bson.M
	if err := cursor.All(ctx, &employees); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for i := range employees {
		delete(employees[i], "password")
	}

	c.JSON(http.StatusOK, employees)
}

func AddCompanyEmployee(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var input struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Récupère l'admin qui fait la demande
	var admin models.User
	err := database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"_id": userID.(string),
	}).Decode(&admin)
	if err != nil || admin.CompanyID == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune société associée"})
		return
	}

	log.Printf("🔍 CompanyID de l'admin: %s", *admin.CompanyID)

	// ✅ Convertir le CompanyID string en ObjectID
	companyOID, err := primitive.ObjectIDFromHex(*admin.CompanyID)
	if err != nil {
		log.Printf("❌ Erreur conversion CompanyID en ObjectID: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ID de société invalide"})
		return
	}

	// Récupère les infos de la company
	var company bson.M
	err = database.MongoCompanyDB.Collection("companies").FindOne(ctx, bson.M{
		"_id": companyOID, // ✅ Utiliser l'ObjectID au lieu du string
	}).Decode(&company)

	if err != nil {
		log.Printf("❌ Société non trouvée: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Société introuvable"})
		return
	}

	log.Printf("✅ Société trouvée: %v", company["name"])

	// Vérifie si l'email existe déjà
	var existing models.User
	err = database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"email": input.Email,
	}).Decode(&existing)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Un compte avec cet email existe déjà"})
		return
	}

	// ✅ Génère un mot de passe aléatoire
	randomPassword := generateRandomPassword(12)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(randomPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur hash mot de passe"})
		return
	}

	// ✅ Crée l'employé avec role "company-customer"
	isAdmin := false
	companyName := ""
	if name, ok := company["name"].(string); ok {
		companyName = name
	}

	newEmployee := models.User{
		ID:             primitive.NewObjectID().Hex(),
		Name:           input.Name,
		Email:          input.Email,
		Password:       string(hashedPassword),
		Role:           "company-customer", // ✅ Rôle company-customer automatiquement
		CompanyID:      admin.CompanyID,
		CompanyName:    companyName,
		IsCompanyAdmin: &isAdmin,
		Provider:       "local",
	}

	_, err = database.MongoAuthDB.Collection("users").InsertOne(ctx, newEmployee)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// ✅ Envoie l'email avec le mot de passe (en arrière-plan)
	go sendEmployeeWelcomeEmail(input.Email, input.Name, companyName, randomPassword)

	log.Printf("✅ Employé créé: %s (%s) pour company %s", input.Name, input.Email, *admin.CompanyID)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Employé ajouté avec succès. Un email avec ses identifiants lui a été envoyé.",
		"id":      newEmployee.ID,
	})
}

func generateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	b := make([]byte, length)
	rand.Read(b)
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

func sendEmployeeWelcomeEmail(email, name, companyName, password string) {
	subject := "Bienvenue chez Cedra - Vos identifiants"

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="fr">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
	<title>Bienvenue chez Cedra</title>
</head>
<body style="font-family: Arial, sans-serif; background-color: #f9f9f9; padding: 20px;">
	<div style="max-width: 600px; margin: auto; background-color: white; padding: 20px; border-radius: 10px;">
		<h2 style="color: #333;">Bienvenue chez Cedra !</h2>
		<p>Bonjour <b>%s</b>,</p>
		<p>Un compte a été créé pour vous sur Cedra par <strong>%s</strong>.</p>

		<h3>Vos identifiants de connexion :</h3>
		<div style="background-color: #f0f0f0; padding: 15px; border-radius: 5px; margin: 20px 0;">
			<p style="margin: 5px 0;"><strong>Email :</strong> %s</p>
			<p style="margin: 5px 0;"><strong>Mot de passe :</strong> <code style="background-color: #e0e0e0; padding: 5px 10px; border-radius: 3px; font-size: 16px;">%s</code></p>
		</div>

		<p>Vous pouvez vous connecter à l'adresse :</p>
		<p style="text-align: center; margin: 20px 0;">
			<a href="https://cedra.eldocam.com/login" style="background-color: #007bff; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Se connecter</a>
		</p>

		<p style="font-size: 14px; color: #888; border-left: 3px solid #ffa500; padding-left: 15px; margin-top: 20px;">
			<strong>⚠️ Sécurité :</strong> Pour des raisons de sécurité, nous vous recommandons vivement de changer votre mot de passe lors de votre première connexion.
		</p>

		<p style="margin-top: 30px; font-size: 14px; color: #888;">
			Si vous avez des questions, n'hésitez pas à nous contacter.
		</p>

		<p style="margin-top: 20px; color: #555;">
			Cordialement,<br>
			<strong>L'équipe Cedra</strong>
		</p>
	</div>
</body>
</html>
	`, name, companyName, email, password)

	// Utilise votre fonction existante (sans PDF)
	err := utils.SendConfirmationEmail(email, subject, htmlBody, nil)

	if err != nil {
		log.Printf("❌ Erreur envoi email à %s: %v", email, err)
	} else {
		log.Printf("✅ Email d'identifiants envoyé à %s", email)
	}
}

// 🟢 DELETE /api/company/employees/:userId
func RemoveCompanyEmployee(c *gin.Context) {
	userID, _ := c.Get("user_id")
	employeeID := c.Param("userId")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var admin models.User
	err := database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"_id": userID.(string),
	}).Decode(&admin)
	if err != nil || admin.CompanyID == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune société associée"})
		return
	}

	var employee models.User
	err = database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"_id": employeeID,
	}).Decode(&employee)
	if err != nil || employee.CompanyID == nil || *employee.CompanyID != *admin.CompanyID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Employé introuvable ou n'appartient pas à votre société"})
		return
	}

	if employeeID == userID.(string) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vous ne pouvez pas vous retirer vous-même"})
		return
	}

	_, err = database.MongoAuthDB.Collection("users").DeleteOne(ctx, bson.M{"_id": employeeID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Employé retiré avec succès"})
}

// 🟢 PUT /api/company/employees/:userId/admin
func ToggleEmployeeAdmin(c *gin.Context) {
	userID, _ := c.Get("user_id")
	employeeID := c.Param("userId")

	var input struct {
		IsCompanyAdmin bool `json:"isCompanyAdmin"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var admin models.User
	err := database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"_id": userID.(string),
	}).Decode(&admin)
	if err != nil || admin.CompanyID == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune société associée"})
		return
	}

	var employee models.User
	err = database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"_id": employeeID,
	}).Decode(&employee)
	if err != nil || employee.CompanyID == nil || *employee.CompanyID != *admin.CompanyID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Employé introuvable ou n'appartient pas à votre société"})
		return
	}

	isAdmin := input.IsCompanyAdmin
	_, err = database.MongoAuthDB.Collection("users").UpdateOne(
		ctx,
		bson.M{"_id": employeeID},
		bson.M{"$set": bson.M{"isCompanyAdmin": &isAdmin}},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Statut admin mis à jour"})
}
