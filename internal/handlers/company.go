package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

//
// --- HANDLERS SOCIÉTÉ ---
//

// 🟢 GET /api/company/me
func GetMyCompany(c *gin.Context) {
	userID, exists := c.Get("user_id")
	log.Printf("🔍 DEBUG /company/me → user_id=%v (exists=%v)", userID, exists)

	if !exists || userID == "" {
		log.Println("❌ Aucun user_id trouvé (JWT invalide ou non transmis)")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non autorisé"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 🔹 Récupérer l'utilisateur pour avoir son companyId
	var user models.User
	userOID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		log.Println("❌ user_id invalide (mauvais format d'ObjectID)")
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID utilisateur invalide"})
		return
	}

	err = database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"_id": userOID,
	}).Decode(&user)

	if err != nil {
		log.Println("❌ Utilisateur introuvable:", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur introuvable"})
		return
	}

	if user.CompanyID == nil || *user.CompanyID == "" {
		log.Println("❌ Aucun companyID associé à l'utilisateur")
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune société associée"})
		return
	}

	// 🔹 Récupérer la société
	companyOID, err := primitive.ObjectIDFromHex(*user.CompanyID)
	if err != nil {
		log.Println("❌ companyID invalide:", *user.CompanyID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID société invalide"})
		return
	}

	var company bson.M
	err = database.MongoCompanyDB.Collection("companies").FindOne(ctx, bson.M{
		"_id": companyOID,
	}).Decode(&company)

	if err != nil {
		log.Println("❌ Société introuvable:", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Société introuvable"})
		return
	}

	log.Printf("✅ Société trouvée pour userID=%v → %v", userID, company["name"])
	c.JSON(http.StatusOK, company)
}

// 🟢 PUT /api/company/billing
func UpdateCompanyBilling(c *gin.Context) {
	userID, _ := c.Get("user_id")
	log.Printf("🧾 UpdateCompanyBilling → user_id=%v", userID)

	var input struct {
		BillingStreet     string `json:"billingStreet"`
		BillingPostalCode string `json:"billingPostalCode"`
		BillingCity       string `json:"billingCity"`
		BillingCountry    string `json:"billingCountry"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		log.Println("❌ JSON invalide:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 🔹 Récupérer l'utilisateur
	var user models.User
	userOID, _ := primitive.ObjectIDFromHex(userID.(string))
	err := database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"_id": userOID,
	}).Decode(&user)

	if err != nil || user.CompanyID == nil {
		log.Println("❌ Aucun companyID pour cet utilisateur")
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune société associée"})
		return
	}

	// 🔹 Mettre à jour la société
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
		log.Println("❌ Erreur mise à jour Mongo:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Adresse de facturation mise à jour"})
}

// 🟢 GET /api/company/employees
func ListCompanyEmployees(c *gin.Context) {
	userID, _ := c.Get("user_id")
	log.Printf("👥 ListCompanyEmployees → user_id=%v", userID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 🔹 Récupérer l'utilisateur
	var user models.User
	userOID, _ := primitive.ObjectIDFromHex(userID.(string))
	err := database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"_id": userOID,
	}).Decode(&user)

	if err != nil || user.CompanyID == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune société associée"})
		return
	}

	// 🔹 Lister tous les utilisateurs de cette société
	cursor, err := database.MongoAuthDB.Collection("users").Find(ctx, bson.M{
		"companyId": *user.CompanyID,
	})
	if err != nil {
		log.Println("❌ Erreur Mongo Find:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération"})
		return
	}
	defer cursor.Close(ctx)

	var employees []bson.M
	if err := cursor.All(ctx, &employees); err != nil {
		log.Println("❌ Erreur décodage employés:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 🔹 Retirer les mots de passe
	for i := range employees {
		delete(employees[i], "password")
	}

	c.JSON(http.StatusOK, employees)
}

// 🟢 POST /api/company/employees
func AddCompanyEmployee(c *gin.Context) {
	userID, _ := c.Get("user_id")
	log.Printf("➕ AddCompanyEmployee → user_id=%v", userID)

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

	// 🔹 Récupérer l'admin
	var admin models.User
	userOID, _ := primitive.ObjectIDFromHex(userID.(string))
	err := database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"_id": userOID,
	}).Decode(&admin)

	if err != nil || admin.CompanyID == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune société associée"})
		return
	}

	// 🔹 Vérifier que l'email n'existe pas déjà
	var existing models.User
	err = database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"email": input.Email,
	}).Decode(&existing)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Un compte avec cet email existe déjà"})
		return
	}

	// 🔹 Hash du mot de passe
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur hash mot de passe"})
		return
	}

	// 🔹 Créer l'employé
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
		log.Println("❌ Erreur Mongo InsertOne:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Employé ajouté avec succès",
		"id":      newEmployee.ID.Hex(),
	})
}

// 🟢 DELETE /api/company/employees/:userId
func RemoveCompanyEmployee(c *gin.Context) {
	userID, _ := c.Get("user_id")
	employeeID := c.Param("userId")
	log.Printf("🗑️ RemoveCompanyEmployee → admin=%v target=%v", userID, employeeID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var admin models.User
	userOID, _ := primitive.ObjectIDFromHex(userID.(string))
	err := database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{"_id": userOID}).Decode(&admin)
	if err != nil || admin.CompanyID == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune société associée"})
		return
	}

	employeeOID, _ := primitive.ObjectIDFromHex(employeeID)
	var employee models.User
	err = database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{"_id": employeeOID}).Decode(&employee)
	if err != nil || employee.CompanyID == nil || *employee.CompanyID != *admin.CompanyID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Employé introuvable ou n'appartient pas à votre société"})
		return
	}

	if employeeID == userID.(string) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vous ne pouvez pas vous retirer vous-même"})
		return
	}

	_, err = database.MongoAuthDB.Collection("users").DeleteOne(ctx, bson.M{"_id": employeeOID})
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
	log.Printf("🔄 ToggleEmployeeAdmin → admin=%v target=%v", userID, employeeID)

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
	userOID, _ := primitive.ObjectIDFromHex(userID.(string))
	err := database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{"_id": userOID}).Decode(&admin)
	if err != nil || admin.CompanyID == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune société associée"})
		return
	}

	employeeOID, _ := primitive.ObjectIDFromHex(employeeID)
	var employee models.User
	err = database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{"_id": employeeOID}).Decode(&employee)
	if err != nil || employee.CompanyID == nil || *employee.CompanyID != *admin.CompanyID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Employé introuvable ou n'appartient pas à votre société"})
		return
	}

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
