package company

import (
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/utils"
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func GetMyCompany(c *gin.Context) {
	userID, exists := c.Get("user_id")
	log.Printf("🔍 DEBUG /company/me → user_id=%v (exists=%v)", userID, exists)

	if !exists || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non autorisé"})
		return
	}

	session, err := database.GetUsersSession()
	if err != nil {
		log.Printf("❌ Erreur session ScyllaDB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	// 🔹 Récupérer l'utilisateur
	userIDStr := fmt.Sprintf("%v", userID)
	uid, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID utilisateur invalide"})
		return
	}
	userUUID := gocql.UUID(uid)

	var (
		email, name, role, provider, providerID, companyName string
		companyID                                            *gocql.UUID
		isCompanyAdmin                                       bool
		createdAt, updatedAt                                 time.Time
	)

	err = session.Query(`SELECT email, name, role, provider, provider_id, company_id, company_name, is_company_admin, created_at, updated_at 
	                     FROM users WHERE user_id = ?`, userUUID).Scan(
		&email, &name, &role, &provider, &providerID, &companyID, &companyName, &isCompanyAdmin, &createdAt, &updatedAt)
	if err != nil {
		log.Printf("❌ Utilisateur introuvable: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur introuvable"})
		return
	}

	if companyID == nil {
		log.Printf("⚠️ Aucune société associée pour l'utilisateur %s", email)
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune société associée"})
		return
	}

	log.Printf("🔍 CompanyID de l'utilisateur: %s", companyID.String())

	// 🔹 Récupérer la société
	var (
		companyNameDB, billingStreet, billingPostalCode, billingCity, billingCountry string
		companyCreatedAt                                                             time.Time
	)

	err = session.Query(`SELECT name, billing_street, billing_postal_code, billing_city, billing_country, created_at 
	                     FROM companies WHERE company_id = ?`, *companyID).Scan(
		&companyNameDB, &billingStreet, &billingPostalCode, &billingCity, &billingCountry, &companyCreatedAt)
	if err != nil {
		log.Printf("❌ Société non trouvée: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Société introuvable"})
		return
	}

	log.Printf("✅ Société trouvée: %s", companyNameDB)

	companyIDStr := companyID.String()
	company := map[string]interface{}{
		"company_id":        companyIDStr,
		"name":              companyNameDB,
		"billingStreet":     billingStreet,
		"billingPostalCode": billingPostalCode,
		"billingCity":       billingCity,
		"billingCountry":    billingCountry,
		"createdAt":         companyCreatedAt,
	}

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

	session, err := database.GetUsersSession()
	if err != nil {
		log.Printf("❌ Erreur session ScyllaDB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	userIDStr := fmt.Sprintf("%v", userID)
	uid, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID utilisateur invalide"})
		return
	}
	userUUID := gocql.UUID(uid)

	var companyID *gocql.UUID
	err = session.Query("SELECT company_id FROM users WHERE user_id = ?", userUUID).Scan(&companyID)
	if err != nil || companyID == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune société associée"})
		return
	}

	// Mettre à jour la société
	err = session.Query(`UPDATE companies SET billing_street = ?, billing_postal_code = ?, billing_city = ?, billing_country = ? 
	                     WHERE company_id = ?`,
		input.BillingStreet, input.BillingPostalCode, input.BillingCity, input.BillingCountry, *companyID).Exec()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Adresse de facturation mise à jour"})
}

func ListCompanyEmployees(c *gin.Context) {
	userID, _ := c.Get("user_id")

	session, err := database.GetUsersSession()
	if err != nil {
		log.Printf("❌ Erreur session ScyllaDB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	userIDStr := fmt.Sprintf("%v", userID)
	uid, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID utilisateur invalide"})
		return
	}
	userUUID := gocql.UUID(uid)

	var companyID *gocql.UUID
	err = session.Query("SELECT company_id FROM users WHERE user_id = ?", userUUID).Scan(&companyID)
	if err != nil || companyID == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune société associée"})
		return
	}

	// Récupérer tous les employés de la société
	var employees []map[string]interface{}
	iter := session.Query(`SELECT user_id, email, name, role, provider, provider_id, company_id, company_name, is_company_admin, created_at, updated_at 
	                       FROM users WHERE company_id = ? ALLOW FILTERING`, *companyID).Iter()
	var (
		empID                                                                  gocql.UUID
		empEmail, empName, empRole, empProvider, empProviderID, empCompanyName string
		empCompanyID                                                           *gocql.UUID
		empIsAdmin                                                             bool
		empCreatedAt, empUpdatedAt                                             time.Time
	)
	for iter.Scan(&empID, &empEmail, &empName, &empRole, &empProvider, &empProviderID, &empCompanyID, &empCompanyName, &empIsAdmin, &empCreatedAt, &empUpdatedAt) {
		var empCompanyIDStr *string
		if empCompanyID != nil {
			s := empCompanyID.String()
			empCompanyIDStr = &s
		}
		employees = append(employees, map[string]interface{}{
			"user_id":        empID.String(),
			"email":          empEmail,
			"name":           empName,
			"role":           empRole,
			"provider":       empProvider,
			"companyId":      empCompanyIDStr,
			"companyName":    empCompanyName,
			"isCompanyAdmin": empIsAdmin,
			"created_at":     empCreatedAt,
			"updated_at":     empUpdatedAt,
		})
	}
	if err := iter.Close(); err != nil {
		log.Printf("⚠️ Erreur fermeture iter: %v", err)
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

	session, err := database.GetUsersSession()
	if err != nil {
		log.Printf("❌ Erreur session ScyllaDB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	userIDStr := fmt.Sprintf("%v", userID)
	uid, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID utilisateur invalide"})
		return
	}
	adminUUID := gocql.UUID(uid)

	// Récupère l'admin qui fait la demande
	var (
		adminEmail, adminName, adminRole, adminProvider, adminProviderID, adminCompanyName string
		adminCompanyID                                                                     *gocql.UUID
		adminIsAdmin                                                                       bool
		adminCreatedAt, adminUpdatedAt                                                     time.Time
	)

	err = session.Query(`SELECT email, name, role, provider, provider_id, company_id, company_name, is_company_admin, created_at, updated_at 
	                     FROM users WHERE user_id = ?`, adminUUID).Scan(
		&adminEmail, &adminName, &adminRole, &adminProvider, &adminProviderID, &adminCompanyID, &adminCompanyName, &adminIsAdmin, &adminCreatedAt, &adminUpdatedAt)
	if err != nil || adminCompanyID == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune société associée"})
		return
	}

	log.Printf("🔍 CompanyID de l'admin: %s", adminCompanyID.String())

	// Récupère les infos de la company
	var companyNameDB, billingStreet, billingPostalCode, billingCity, billingCountry string
	var companyCreatedAt time.Time

	err = session.Query(`SELECT name, billing_street, billing_postal_code, billing_city, billing_country, created_at 
	                     FROM companies WHERE company_id = ?`, *adminCompanyID).Scan(
		&companyNameDB, &billingStreet, &billingPostalCode, &billingCity, &billingCountry, &companyCreatedAt)
	if err != nil {
		log.Printf("❌ Société non trouvée: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Société introuvable"})
		return
	}

	log.Printf("✅ Société trouvée: %s", companyNameDB)

	// Vérifie si l'email existe déjà
	var existingUserID gocql.UUID
	err = session.Query("SELECT user_id FROM users_by_email WHERE email = ?", input.Email).Scan(&existingUserID)
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
	employeeID := gocql.TimeUUID()
	employeeIDStr := employeeID.String()
	now := time.Now()

	// Insert dans users
	err = session.Query(`INSERT INTO users (user_id, email, password, name, role, provider, provider_id, company_id, company_name, is_company_admin, created_at, updated_at) 
	                     VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		employeeID, input.Email, string(hashedPassword), input.Name, "company-customer", "local", "", *adminCompanyID, companyNameDB, isAdmin, now, now).Exec()
	if err != nil {
		log.Printf("❌ Erreur insertion employé: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Insert dans users_by_email
	err = session.Query("INSERT INTO users_by_email (email, user_id) VALUES (?, ?)", input.Email, employeeID).Exec()
	if err != nil {
		log.Printf("⚠️ Erreur insertion index email: %v", err)
	}

	// ✅ Envoie l'email avec le mot de passe (en arrière-plan)
	go sendEmployeeWelcomeEmail(input.Email, input.Name, companyNameDB, randomPassword)

	log.Printf("✅ Employé créé: %s (%s) pour company %s", input.Name, input.Email, adminCompanyID.String())

	c.JSON(http.StatusCreated, gin.H{
		"message": "Employé ajouté avec succès. Un email avec ses identifiants lui a été envoyé.",
		"id":      employeeIDStr,
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

	session, err := database.GetUsersSession()
	if err != nil {
		log.Printf("❌ Erreur session ScyllaDB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	userIDStr := fmt.Sprintf("%v", userID)
	adminUID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID utilisateur invalide"})
		return
	}
	adminUUID := gocql.UUID(adminUID)

	// Récupérer l'admin
	var adminCompanyID *gocql.UUID
	err = session.Query("SELECT company_id FROM users WHERE user_id = ?", adminUUID).Scan(&adminCompanyID)
	if err != nil || adminCompanyID == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune société associée"})
		return
	}

	// Récupérer l'employé
	employeeUID, err := uuid.Parse(employeeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID employé invalide"})
		return
	}
	employeeUUID := gocql.UUID(employeeUID)

	var employeeCompanyID *gocql.UUID
	err = session.Query("SELECT company_id FROM users WHERE user_id = ?", employeeUUID).Scan(&employeeCompanyID)
	if err != nil || employeeCompanyID == nil || *employeeCompanyID != *adminCompanyID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Employé introuvable ou n'appartient pas à votre société"})
		return
	}

	if employeeID == userIDStr {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vous ne pouvez pas vous retirer vous-même"})
		return
	}

	// Récupérer l'email avant de supprimer l'utilisateur
	var employeeEmail string
	err = session.Query("SELECT email FROM users WHERE user_id = ?", employeeUUID).Scan(&employeeEmail)
	if err != nil {
		log.Printf("⚠️ Impossible de récupérer l'email de l'employé: %v", err)
	}

	// Supprimer l'employé
	err = session.Query("DELETE FROM users WHERE user_id = ?", employeeUUID).Exec()
	if err != nil {
		log.Printf("❌ Erreur suppression employé: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression"})
		return
	}

	// Supprimer aussi de l'index email
	if employeeEmail != "" {
		session.Query("DELETE FROM users_by_email WHERE email = ?", employeeEmail).Exec()
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

	session, err := database.GetUsersSession()
	if err != nil {
		log.Printf("❌ Erreur session ScyllaDB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	userIDStr := fmt.Sprintf("%v", userID)
	adminUID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID utilisateur invalide"})
		return
	}
	adminUUID := gocql.UUID(adminUID)

	// Récupérer l'admin
	var adminCompanyID *gocql.UUID
	err = session.Query("SELECT company_id FROM users WHERE user_id = ?", adminUUID).Scan(&adminCompanyID)
	if err != nil || adminCompanyID == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune société associée"})
		return
	}

	// Récupérer l'employé
	employeeUID, err := uuid.Parse(employeeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID employé invalide"})
		return
	}
	employeeUUID := gocql.UUID(employeeUID)

	var employeeCompanyID *gocql.UUID
	err = session.Query("SELECT company_id FROM users WHERE user_id = ?", employeeUUID).Scan(&employeeCompanyID)
	if err != nil || employeeCompanyID == nil || *employeeCompanyID != *adminCompanyID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Employé introuvable ou n'appartient pas à votre société"})
		return
	}

	// Mettre à jour le statut admin
	now := time.Now()
	err = session.Query("UPDATE users SET is_company_admin = ?, updated_at = ? WHERE user_id = ?",
		input.IsCompanyAdmin, now, employeeUUID).Exec()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Statut admin mis à jour"})
}
