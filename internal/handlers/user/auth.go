package user

import (
	"context"
	"net/http"
	"os"
	"time"
	"log"
	"strings" // ✅ AJOUT
	
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/markbates/goth/gothic"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo" // ✅ AJOUT
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

// ================== AUTH LOCALE ==================

func CreateUser(c *gin.Context) {
	var input struct {
		Name              string `json:"name"`
		Email             string `json:"email"`
		Password          string `json:"password"`
		IsCompanyAdmin    bool   `json:"isCompanyAdmin"`
		CompanyName       string `json:"companyName"`
		BillingStreet     string `json:"billingStreet"`
		BillingPostalCode string `json:"billingPostalCode"`
		BillingCity       string `json:"billingCity"`
		BillingCountry    string `json:"billingCountry"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Vérifier si l'email existe déjà
	var existing models.User
	err := database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"email":    input.Email,
		"provider": "local",
	}).Decode(&existing)

	if err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Un compte avec cet email existe déjà",
			"email": input.Email,
		})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur hash mot de passe"})
		return
	}

	var companyID *string

	// Si admin société, créer l'entreprise
	if input.IsCompanyAdmin {
		// Validation des champs requis
		if input.CompanyName == "" || input.BillingStreet == "" ||
			input.BillingPostalCode == "" || input.BillingCity == "" ||
			input.BillingCountry == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Tous les champs de facturation sont requis pour un compte société",
			})
			return
		}

		// Créer la société
		company := bson.M{
			"name":              input.CompanyName,
			"billingStreet":     input.BillingStreet,
			"billingPostalCode": input.BillingPostalCode,
			"billingCity":       input.BillingCity,
			"billingCountry":    input.BillingCountry,
			"createdAt":         primitive.NewDateTimeFromTime(time.Now()),
		}

		result, err := database.MongoCompanyDB.Collection("companies").InsertOne(ctx, company)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création de la société"})
			return
		}

		companyIDHex := result.InsertedID.(primitive.ObjectID).Hex()
		companyID = &companyIDHex
	}

	// Créer l'utilisateur
	newUser := models.User{
		ID:             primitive.NewObjectID(),
		Name:           input.Name,
		Email:          input.Email,
		Password:       string(hashedPassword),
		Role:           "customer",
		IsCompanyAdmin: input.IsCompanyAdmin,
		Provider:       "local",
	}

	if companyID != nil {
		newUser.CompanyID = companyID
	}

	_, err = database.MongoAuthDB.Collection("users").InsertOne(ctx, newUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	token := generateJWT(newUser)

	c.JSON(http.StatusCreated, gin.H{
		"token":          token,
		"id":             newUser.ID.Hex(),
		"email":          newUser.Email,
		"role":           newUser.Role,
		"isCompanyAdmin": newUser.IsCompanyAdmin,
		"companyId":      companyID,
	})
}

func Login(c *gin.Context) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"email":    input.Email,
		"provider": "local",
	}).Decode(&user)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur introuvable"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Mot de passe incorrect"})
		return
	}

	// Récupérer les infos de la société si applicable
	var companyName *string
	if user.CompanyID != nil && *user.CompanyID != "" {
		var company struct {
			Name string `bson:"name"`
		}

		companyOID, err := primitive.ObjectIDFromHex(*user.CompanyID)
		if err == nil {
			err := database.MongoCompanyDB.Collection("companies").FindOne(
				ctx,
				bson.M{"_id": companyOID},
			).Decode(&company)

			if err == nil {
				companyName = &company.Name
			}
		}
	}

	token := generateJWT(user)
	c.JSON(http.StatusOK, gin.H{
		"token":          token,
		"userId":         user.ID.Hex(),
		"name":           user.Name,
		"email":          user.Email,
		"role":           user.Role,
		"companyId":      user.CompanyID,
		"companyName":    companyName,
		"isCompanyAdmin": user.IsCompanyAdmin,
	})
}

// ================== AUTH SOCIALE ==================

func BeginAuth(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "aucun provider spécifié"})
		return
	}

	// ✅ Récupère et sauvegarde le redirect_uri
	redirectURI := c.Query("redirect_uri")
	if redirectURI != "" {
		session, _ := gothic.Store.Get(c.Request, "goth_session")
		session.Values["redirect_uri"] = redirectURI
		session.Save(c.Request, c.Writer)
	}

	q := c.Request.URL.Query()
	q.Set("provider", provider)
	c.Request.URL.RawQuery = q.Encode()

	log.Printf("🔐 BeginAuth pour provider: %s", provider)
	gothic.BeginAuthHandler(c.Writer, c.Request)
}

func CallbackAuth(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "aucun provider spécifié"})
		return
	}

	q := c.Request.URL.Query()
	q.Set("provider", provider)
	c.Request.URL.RawQuery = q.Encode()

	log.Printf("🔐 CallbackAuth pour provider: %s", provider)

	userInfo, err := gothic.CompleteUserAuth(c.Writer, c.Request)
	if err != nil {
		log.Printf("❌ Erreur CompleteUserAuth: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("✅ Auth réussie pour %s via %s", userInfo.Email, provider)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := database.MongoAuthDB.Collection("users")
	var user models.User
	var isNewUser bool

	// ✅ ÉTAPE 1 : Chercher par provider + provider_id (utilisateur OAuth existant)
	err = collection.FindOne(ctx, bson.M{
		"provider":    provider,
		"provider_id": userInfo.UserID,
	}).Decode(&user)

	switch err {
	case nil:
		// ✅ Utilisateur OAuth trouvé
		log.Printf("✅ Utilisateur OAuth existant trouvé: %s", user.Email)
	case mongo.ErrNoDocuments:
		// ✅ ÉTAPE 2 : Chercher par email (compte local ou autre provider)
		err = collection.FindOne(ctx, bson.M{
			"email": userInfo.Email,
		}).Decode(&user)

		if err == nil {
			// ✅ FUSION : Un compte avec cet email existe déjà
			log.Printf("🔗 Fusion de compte : %s (%s) → %s", user.Email, user.Provider, provider)

			update := bson.M{
				"$set": bson.M{
					"provider":    provider,
					"provider_id": userInfo.UserID,
					"name":        userInfo.Name,
				},
			}

			_, err := collection.UpdateOne(ctx, bson.M{"_id": user.ID}, update)
			if err != nil {
				log.Printf("❌ Erreur mise à jour compte: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur fusion compte"})
				return
			}

			// Recharger l'utilisateur
			err = collection.FindOne(ctx, bson.M{"_id": user.ID}).Decode(&user)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur rechargement utilisateur"})
				return
			}

			log.Printf("✅ Compte fusionné avec succès")
		} else {
			// ✅ ÉTAPE 3 : Créer un nouveau compte
			isNewUser = true
			user = models.User{
				ID:         primitive.NewObjectID(),
				Email:      userInfo.Email,
				Name:       userInfo.Name,
				Provider:   provider,
				ProviderID: userInfo.UserID,
				Role:       "customer",
			}

			_, err := collection.InsertOne(ctx, user)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur enregistrement utilisateur"})
				return
			}
			log.Printf("✅ Nouvel utilisateur créé: %s", user.Email)
		}
	default:
		// Autre erreur MongoDB
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur base de données"})
		return
	}

	token := generateJWT(user)

	// ✅ Récupère le redirect_uri depuis la session
	session, _ := gothic.Store.Get(c.Request, "goth_session")
	redirectURI, ok := session.Values["redirect_uri"].(string)
	
	if !ok || redirectURI == "" {
		// Fallback sur l'env
		redirectURI = os.Getenv("FRONTEND_URL")
		if redirectURI == "" {
			redirectURI = "http://localhost:5173"
		}
	}

	// ✅ Valide que le redirect_uri est autorisé
	allowedOrigins := []string{
		"http://localhost:5173",
		"http://localhost:3000",
		"https://cedra.com",
		"cedra://auth/callback",
		"myapp://auth/callback",
	}

	isAllowed := false
	for _, origin := range allowedOrigins {
		if strings.HasPrefix(redirectURI, origin) {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		log.Printf("⚠️ Redirect URI non autorisé: %s", redirectURI)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Redirect URI non autorisé"})
		return
	}

	// ✅ Construit l'URL de redirection
	separator := "?"
	if strings.Contains(redirectURI, "?") {
		separator = "&"
	}

	finalURL := redirectURI + separator + "token=" + token
	if isNewUser {
		finalURL += "&new_user=true"
	}

	log.Printf("✅ Redirection vers: %s", finalURL)
	c.Redirect(http.StatusTemporaryRedirect, finalURL)
}

func MergeAccount(c *gin.Context) {
	var input struct {
		Email      string `json:"email"`
		Provider   string `json:"provider"`
		ProviderID string `json:"provider_id"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := database.MongoAuthDB.Collection("users")

	objID, _ := primitive.ObjectIDFromHex(userID)
	var user models.User
	err := collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&user)
	if err != nil || user.Email != input.Email {
		c.JSON(http.StatusForbidden, gin.H{"error": "Email non autorisé"})
		return
	}

	update := bson.M{
		"$set": bson.M{
			"provider":    input.Provider,
			"provider_id": input.ProviderID,
		},
	}

	_, err = collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur fusion"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Comptes fusionnés avec succès"})
}

// ================== UTILITAIRES ==================

func generateJWT(user models.User) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":        user.ID.Hex(),
		"email":          user.Email,
		"role":           user.Role,
		"isCompanyAdmin": user.IsCompanyAdmin,
		"exp":            time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, _ := token.SignedString(jwtSecret)
	return tokenString
}

func Me(c *gin.Context) {
	userID, _ := c.Get("user_id")
	email, _ := c.Get("email")
	role, _ := c.Get("role")
	isCompanyAdmin, _ := c.Get("isCompanyAdmin")
	name, _ := c.Get("name")

	c.JSON(http.StatusOK, gin.H{
		"user_id":        userID,
		"email":          email,
		"role":           role,
		"name":           name,
		"isCompanyAdmin": isCompanyAdmin,
	})
}