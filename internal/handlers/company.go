package handlers

import (
	"context"
	"net/http"
	"time"
	"golang.org/x/crypto/bcrypt"
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetMyCompany récupère les infos de la société de l'utilisateur connecté
func GetMyCompany(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non autorisé"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Récupérer l'utilisateur pour avoir son companyId
	var user models.User
	userOID, _ := primitive.ObjectIDFromHex(userID.(string))
	err := database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"_id": userOID,
	}).Decode(&user)

	if err != nil || user.CompanyID == nil || *user.CompanyID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune société associée"})
		return
	}

	// Récupérer la société
	companyOID, _ := primitive.ObjectIDFromHex(*user.CompanyID)
	var company bson.M

	err = database.MongoCompanyDB.Collection("companies").FindOne(ctx, bson.M{
		"_id": companyOID,
	}).Decode(&company)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Société introuvable"})
		return
	}

	c.JSON(http.StatusOK, company)
}

// UpdateCompanyBilling met à jour l'adresse de facturation (admin uniquement)
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

	// Récupérer l'utilisateur
	var user models.User
	userOID, _ := primitive.ObjectIDFromHex(userID.(string))
	err := database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"_id": userOID,
	}).Decode(&user)

	if err != nil || user.CompanyID == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune société associée"})
		return
	}

	// Mettre à jour la société
	companyOID, _ := primitive.ObjectIDFromHex(*user.CompanyID)
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
		bson.M{"_id": companyOID},
		update,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Adresse de facturation mise à jour"})
}

// ListCompanyEmployees liste tous les employés de la société
func ListCompanyEmployees(c *gin.Context) {
	userID, _ := c.Get("user_id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Récupérer l'utilisateur
	var user models.User
	userOID, _ := primitive.ObjectIDFromHex(userID.(string))
	err := database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"_id": userOID,
	}).Decode(&user)

	if err != nil || user.CompanyID == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune société associée"})
		return
	}

	// Lister tous les utilisateurs de cette société
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

	// Retirer les mots de passe
	for i := range employees {
		delete(employees[i], "password")
	}

	c.JSON(http.StatusOK, employees)
}

// AddCompanyEmployee ajoute un employé à la société
func AddCompanyEmployee(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Récupérer l'admin
	var admin models.User
	userOID, _ := primitive.ObjectIDFromHex(userID.(string))
	err := database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"_id": userOID,
	}).Decode(&admin)

	if err != nil || admin.CompanyID == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune société associée"})
		return
	}

	// Vérifier que l'email n'existe pas déjà
	var existing models.User
	err = database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"email": input.Email,
	}).Decode(&existing)

	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Un compte avec cet email existe déjà"})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur hash mot de passe"})
		return
	}

	// Créer l'employé
	newEmployee := models.User{
		ID:             primitive.NewObjectID(),
		Name:           input.Name,
		Email:          input.Email,
		Password:       string(hashedPassword),
		Role:           "customer",
		CompanyID:      admin.CompanyID,
		IsCompanyAdmin: false,
		Provider:       "local",
	}

	_, err = database.MongoAuthDB.Collection("users").InsertOne(ctx, newEmployee)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Employé ajouté avec succès",
		"id":      newEmployee.ID.Hex(),
	})
}

// RemoveCompanyEmployee retire un employé de la société
func RemoveCompanyEmployee(c *gin.Context) {
	userID, _ := c.Get("user_id")
	employeeID := c.Param("userId")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Récupérer l'admin
	var admin models.User
	userOID, _ := primitive.ObjectIDFromHex(userID.(string))
	err := database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"_id": userOID,
	}).Decode(&admin)

	if err != nil || admin.CompanyID == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune société associée"})
		return
	}

	// Vérifier que l'employé appartient à la même société
	employeeOID, _ := primitive.ObjectIDFromHex(employeeID)
	var employee models.User
	err = database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"_id": employeeOID,
	}).Decode(&employee)

	if err != nil || employee.CompanyID == nil || *employee.CompanyID != *admin.CompanyID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Employé introuvable ou n'appartient pas à votre société"})
		return
	}

	// Empêcher de se supprimer soi-même
	if employeeID == userID.(string) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vous ne pouvez pas vous retirer vous-même"})
		return
	}

	// Supprimer l'employé
	_, err = database.MongoAuthDB.Collection("users").DeleteOne(ctx, bson.M{
		"_id": employeeOID,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Employé retiré avec succès"})
}

// ToggleEmployeeAdmin bascule le statut admin d'un employé
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

	// Récupérer l'admin
	var admin models.User
	userOID, _ := primitive.ObjectIDFromHex(userID.(string))
	err := database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"_id": userOID,
	}).Decode(&admin)

	if err != nil || admin.CompanyID == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune société associée"})
		return
	}

	// Vérifier que l'employé appartient à la même société
	employeeOID, _ := primitive.ObjectIDFromHex(employeeID)
	var employee models.User
	err = database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"_id": employeeOID,
	}).Decode(&employee)

	if err != nil || employee.CompanyID == nil || *employee.CompanyID != *admin.CompanyID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Employé introuvable ou n'appartient pas à votre société"})
		return
	}

	// Mettre à jour le statut admin
	_, err = database.MongoAuthDB.Collection("users").UpdateOne(
		ctx,
		bson.M{"_id": employeeOID},
		bson.M{"$set": bson.M{"isCompanyAdmin": input.IsCompanyAdmin}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Statut admin mis à jour"})
}