package company

import (
	"context"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"

	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

//
// --- HANDLERS SOCIÉTÉ ---
//

// 🟢 GET /api/company/me
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
		c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur introuvable"})
		return
	}

	if user.CompanyID == nil || *user.CompanyID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune société associée"})
		return
	}

	// 🔹 Récupérer la société
	var company bson.M
	err = database.MongoCompanyDB.Collection("companies").FindOne(ctx, bson.M{
		"_id": *user.CompanyID,
	}).Decode(&company)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Société introuvable"})
		return
	}

	c.JSON(http.StatusOK, company)
}

// 🟢 PUT /api/company/billing
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
		bson.M{"_id": *user.CompanyID},
		update,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Adresse de facturation mise à jour"})
}

// 🟢 GET /api/company/employees
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

// 🟢 POST /api/company/employees
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

	var admin models.User
	err := database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"_id": userID.(string),
	}).Decode(&admin)
	if err != nil || admin.CompanyID == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune société associée"})
		return
	}

	var existing models.User
	err = database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"email": input.Email,
	}).Decode(&existing)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Un compte avec cet email existe déjà"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur hash mot de passe"})
		return
	}

	isAdmin := false
	newEmployee := models.User{
		ID:             primitive.NewObjectID().Hex(),
		Name:           input.Name,
		Email:          input.Email,
		Password:       string(hashedPassword),
		Role:           "customer",
		CompanyID:      admin.CompanyID,
		IsCompanyAdmin: &isAdmin,
		Provider:       "local",
	}

	_, err = database.MongoAuthDB.Collection("users").InsertOne(ctx, newEmployee)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Employé ajouté avec succès",
		"id":      newEmployee.ID,
	})
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
