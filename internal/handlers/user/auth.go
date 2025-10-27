package user

import (
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/facebook"
	"github.com/markbates/goth/providers/google"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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

	// email déjà pris pour un compte local ?
	var existing models.User
	err := database.MongoAuthDB.Collection("users").
		FindOne(ctx, bson.M{"email": input.Email, "provider": "local"}).
		Decode(&existing)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Un compte avec cet email existe déjà"})
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)

	// crée la company si admin
	var companyID *string
	if input.IsCompanyAdmin {
		company := bson.M{
			"name":              input.CompanyName,
			"billingStreet":     input.BillingStreet,
			"billingPostalCode": input.BillingPostalCode,
			"billingCity":       input.BillingCity,
			"billingCountry":    input.BillingCountry,
			"createdAt":         primitive.NewDateTimeFromTime(time.Now()),
		}
		result, _ := database.MongoCompanyDB.Collection("companies").InsertOne(ctx, company)
		if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
			h := oid.Hex()
			companyID = &h
		}
	}

	// ✅ Définit le rôle selon le type de compte dès l'inscription
	role := "customer"
	if input.IsCompanyAdmin {
		role = "company-customer"
	}

	id := primitive.NewObjectID().Hex()
	isAdmin := input.IsCompanyAdmin
	user := models.User{
		ID:             id,
		Name:           input.Name,
		Email:          input.Email,
		Password:       string(hashedPassword),
		Role:           role, // ✅ Rôle défini dès la création
		Provider:       "local",
		IsCompanyAdmin: &isAdmin,
		CompanyID:      companyID,
		CompanyName:    input.CompanyName,
	}

	if _, err := database.MongoAuthDB.Collection("users").InsertOne(ctx, user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur création utilisateur"})
		return
	}

	token := generateJWT(user)
	c.JSON(http.StatusCreated, gin.H{
		"token":          token,
		"userId":         user.ID,
		"email":          user.Email,
		"name":           user.Name,
		"role":           user.Role,
		"isCompanyAdmin": user.IsCompanyAdmin,
		"companyId":      user.CompanyID,
		"companyName":    user.CompanyName,
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err := database.MongoAuthDB.Collection("users").
		FindOne(ctx, bson.M{"email": input.Email, "provider": "local"}).
		Decode(&user)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email ou mot de passe incorrect"})
		return
	}

	token := generateJWT(user)
	c.JSON(http.StatusOK, gin.H{
		"token":          token,
		"userId":         user.ID,
		"email":          user.Email,
		"name":           user.Name,
		"role":           user.Role,
		"isCompanyAdmin": user.IsCompanyAdmin,
		"companyId":      user.CompanyID,
		"companyName":    user.CompanyName,
	})
}

// ================== AUTH SOCIALE (WEB) ==================

func BeginAuth(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "aucun provider spécifié"})
		return
	}

	redirectURL := c.Query("redirect_url")
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	callbackURL := baseURL + "/api/auth/" + provider + "/callback"

	switch provider {
	case "google":
		goth.UseProviders(google.New(
			os.Getenv("GOOGLE_CLIENT_ID"),
			os.Getenv("GOOGLE_CLIENT_SECRET"),
			callbackURL,
		))
	case "facebook":
		goth.UseProviders(facebook.New(
			os.Getenv("FACEBOOK_CLIENT_ID"),
			os.Getenv("FACEBOOK_CLIENT_SECRET"),
			callbackURL,
		))
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provider non supporté"})
		return
	}

	ctx := context.Background()
	state := generateRandomState()
	if redirectURL != "" {
		_ = database.RedisClient.Set(ctx, "oauth_redirect:"+state, redirectURL, 10*time.Minute).Err()
	}

	q := c.Request.URL.Query()
	q.Set("provider", provider)
	q.Set("state", state)
	c.Request.URL.RawQuery = q.Encode()
	gothic.BeginAuthHandler(c.Writer, c.Request)
}

func generateRandomState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func CallbackAuth(c *gin.Context) {
	provider := c.Param("provider")
	state := c.Query("state")
	code := c.Query("code")
	if provider == "" || code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Paramètres OAuth invalides"})
		return
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	var userEmail, userName, userID string

	switch provider {
	case "google":
		clientID := os.Getenv("GOOGLE_CLIENT_ID")
		clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
		redirect := baseURL + "/api/auth/google/callback"

		data := url.Values{}
		data.Set("code", code)
		data.Set("client_id", clientID)
		data.Set("client_secret", clientSecret)
		data.Set("redirect_uri", redirect)
		data.Set("grant_type", "authorization_code")

		resp, _ := http.PostForm("https://oauth2.googleapis.com/token", data)
		defer resp.Body.Close()
		var tokenResp struct {
			AccessToken string `json:"access_token"`
		}
		json.NewDecoder(resp.Body).Decode(&tokenResp)

		userResp, _ := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + tokenResp.AccessToken)
		defer userResp.Body.Close()
		var gu struct{ ID, Email, Name string }
		json.NewDecoder(userResp.Body).Decode(&gu)
		userID, userEmail, userName = gu.ID, gu.Email, gu.Name

	case "facebook":
		clientID := os.Getenv("FACEBOOK_CLIENT_ID")
		clientSecret := os.Getenv("FACEBOOK_CLIENT_SECRET")
		redirect := baseURL + "/api/auth/facebook/callback"

		tokenURL := fmt.Sprintf("https://graph.facebook.com/v12.0/oauth/access_token?client_id=%s&redirect_uri=%s&client_secret=%s&code=%s",
			clientID, url.QueryEscape(redirect), clientSecret, code)
		resp, _ := http.Get(tokenURL)
		defer resp.Body.Close()
		var tokenResp struct{ AccessToken string }
		json.NewDecoder(resp.Body).Decode(&tokenResp)

		userResp, _ := http.Get("https://graph.facebook.com/me?fields=id,name,email&access_token=" + tokenResp.AccessToken)
		defer userResp.Body.Close()
		var fb struct{ ID, Email, Name string }
		json.NewDecoder(userResp.Body).Decode(&fb)
		userID, userEmail, userName = fb.ID, fb.Email, fb.Name
	}

	handleOAuthUser(c, provider, userID, userEmail, userName, state)
}

// ================== AUTH SOCIALE (MOBILE) ==================

func GoogleMobileLogin(c *gin.Context) {
	var body struct {
		IDToken string `json:"id_token"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.IDToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id_token manquant"})
		return
	}

	clientIDs := []string{
		os.Getenv("GOOGLE_WEB_CLIENT_ID"),
		os.Getenv("GOOGLE_IOS_CLIENT_ID"),
		os.Getenv("GOOGLE_ANDROID_CLIENT_ID"),
	}

	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + url.QueryEscape(body.IDToken))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur vérification Google"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token Google invalide"})
		return
	}

	var payload struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Audience string `json:"aud"`
		Subject  string `json:"sub"`
	}
	json.NewDecoder(resp.Body).Decode(&payload)

	valid := false
	for _, id := range clientIDs {
		if payload.Audience == id && id != "" {
			valid = true
			break
		}
	}
	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Client ID non autorisé"})
		return
	}

	user := findOrCreateOAuthUser("google", payload.Subject, payload.Email, payload.Name)
	token := generateJWT(user)
	c.JSON(http.StatusOK, gin.H{"token": token, "email": user.Email, "name": user.Name})
}

func FacebookMobileLogin(c *gin.Context) {
	var body struct {
		AccessToken string `json:"access_token"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.AccessToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "access_token manquant"})
		return
	}

	resp, err := http.Get("https://graph.facebook.com/me?fields=id,name,email&access_token=" + body.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur API Facebook"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token Facebook invalide"})
		return
	}

	var fb struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	json.NewDecoder(resp.Body).Decode(&fb)

	user := findOrCreateOAuthUser("facebook", fb.ID, fb.Email, fb.Name)
	token := generateJWT(user)
	c.JSON(http.StatusOK, gin.H{"token": token, "email": user.Email, "name": user.Name})
}

// ================== UTILITAIRES ==================

func findOrCreateOAuthUser(provider, providerID, email, name string) models.User {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	col := database.MongoAuthDB.Collection("users")
	var user models.User

	// 1️⃣ Recherche par provider_id
	err := col.FindOne(ctx, bson.M{"provider": provider, "provider_id": providerID}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		// 2️⃣ Sinon, recherche par email
		err = col.FindOne(ctx, bson.M{"email": email}).Decode(&user)
		if err == mongo.ErrNoDocuments {
			// 3️⃣ Création d'un nouvel utilisateur OAuth avec rôle "pending"
			id := primitive.NewObjectID().Hex()
			isAdmin := false
			user = models.User{
				ID:             id,
				Email:          email,
				Name:           name,
				Provider:       provider,
				ProviderID:     providerID,
				Role:           "pending", // ✅ Rôle temporaire jusqu'à completion du profil
				IsCompanyAdmin: &isAdmin,
			}
			_, _ = col.InsertOne(ctx, user)
			log.Printf("🆕 Utilisateur OAuth créé (%s) avec rôle pending : %s", provider, email)
		} else {
			// 4️⃣ Si utilisateur existant → on met à jour son provider
			_, _ = col.UpdateOne(ctx, bson.M{"email": email}, bson.M{
				"$set": bson.M{
					"provider":    provider,
					"provider_id": providerID,
					"name":        name,
				},
			})
			log.Printf("🔄 Compte existant fusionné avec provider %s : %s", provider, email)
		}
	} else {
		log.Printf("✅ Utilisateur OAuth existant trouvé : %s", email)
	}

	return user
}

func handleOAuthUser(c *gin.Context, provider, providerID, email, name, state string) {
	ctx := context.Background()
	user := findOrCreateOAuthUser(provider, providerID, email, name)
	token := generateJWT(user)

	redirectURI, _ := database.RedisClient.Get(ctx, "oauth_redirect:"+state).Result()
	_, _ = database.RedisClient.Del(ctx, "oauth_redirect:"+state).Result()

	if redirectURI == "" {
		redirectURI = os.Getenv("FRONTEND_URL")
		if redirectURI == "" {
			redirectURI = "http://localhost:5173"
		}
	}

	// iOS deep link auto si user-agent iOS
	if !strings.HasPrefix(redirectURI, "cedra://") {
		ua := strings.ToLower(c.Request.UserAgent())
		if strings.Contains(ua, "iphone") || strings.Contains(ua, "ios") || strings.Contains(ua, "mobile") {
			if v := os.Getenv("IOS_REDIRECT_URL"); v != "" {
				redirectURI = v
			} else {
				redirectURI = "cedra://auth/callback"
			}
		}
	}

	allowed := []string{
		"http://localhost:5173",
		"http://localhost:3000",
		"http://cedra.eldocam.com:5173",
		"http://cedra.eldocam.com",
		"http://cedra.eldocam.com:8080",
		"https://cedra.eldocam.com",
		"https://cedra.com",
		"cedra://auth/callback",
	}
	valid := false
	for _, o := range allowed {
		if strings.HasPrefix(redirectURI, o) {
			valid = true
			break
		}
	}
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Redirect url non autorisé"})
		return
	}

	sep := "?"
	if strings.Contains(redirectURI, "?") {
		sep = "&"
	}
	final := redirectURI + sep + "token=" + url.QueryEscape(token)
	log.Printf("✅ Redirection finale: %s", final)
	c.Redirect(http.StatusTemporaryRedirect, final)
}

func generateJWT(user models.User) string {
	// *bool -> bool (default false si nil)
	isAdmin := false
	if user.IsCompanyAdmin != nil {
		isAdmin = *user.IsCompanyAdmin
	}

	log.Printf("🔧 Génération JWT pour user.ID=%s, email=%s", user.ID, user.Email) // ✅ Ajout log

	claims := jwt.MapClaims{
		"user_id":        user.ID, // ✅ Vérifiez que user.ID n'est pas vide
		"email":          user.Email,
		"role":           user.Role,
		"isCompanyAdmin": isAdmin,
		"exp":            time.Now().Add(24 * time.Hour).Unix(),
	}

	log.Printf("🔧 Claims générés: %+v", claims) // ✅ Ajout log

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(jwtSecret)

	log.Printf("✅ JWT généré (20 premiers chars): %s...", tokenString[:min(20, len(tokenString))]) // ✅ Ajout log

	return tokenString
}

// ================== HANDLERS SUPPLÉMENTAIRES ==================

func CompleteProfile(c *gin.Context) {
	log.Println("🎯 CompleteProfile appelé")
	log.Printf("🔐 Headers reçus: %+v", c.Request.Header)

	userID, ok := c.Get("user_id")
	if !ok {
		log.Println("❌ user_id non trouvé dans context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur non authentifié"})
		return
	}

	log.Printf("✅ user_id trouvé: %v", userID)

	var input struct {
		Name              string `json:"name"`
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

	id := fmt.Sprintf("%v", userID)

	// Crée la company si admin
	var companyID *string
	if input.IsCompanyAdmin && input.CompanyName != "" {
		company := bson.M{
			"name":              input.CompanyName,
			"billingStreet":     input.BillingStreet,
			"billingPostalCode": input.BillingPostalCode,
			"billingCity":       input.BillingCity,
			"billingCountry":    input.BillingCountry,
			"createdAt":         primitive.NewDateTimeFromTime(time.Now()),
		}
		result, err := database.MongoCompanyDB.Collection("companies").InsertOne(ctx, company)
		if err == nil {
			if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
				h := oid.Hex()
				companyID = &h
				log.Printf("✅ Company créée: %s", h)
			}
		} else {
			log.Printf("❌ Erreur création company: %v", err)
		}
	}

	// ✅ Définit le rôle selon le type de compte
	role := "customer" // Par défaut : client particulier
	if input.IsCompanyAdmin {
		role = "company-customer" // Client professionnel
	}

	// Met à jour l'utilisateur
	update := bson.M{
		"$set": bson.M{
			"name":           input.Name,
			"isCompanyAdmin": input.IsCompanyAdmin,
			"companyName":    input.CompanyName,
			"role":           role, // ✅ Mise à jour du rôle
		},
	}

	if companyID != nil {
		update["$set"].(bson.M)["companyId"] = *companyID
	}

	col := database.MongoAuthDB.Collection("users")
	_, err := col.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		// Essaie avec ObjectID si la string ne marche pas
		if oid, e := primitive.ObjectIDFromHex(id); e == nil {
			_, err = col.UpdateOne(ctx, bson.M{"_id": oid}, update)
		}
	}

	if err != nil {
		log.Printf("❌ Erreur mise à jour: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur mise à jour profil"})
		return
	}

	log.Printf("✅ Profil complété pour user %s (role: %s, isCompanyAdmin: %v)", id, role, input.IsCompanyAdmin)

	c.JSON(http.StatusOK, gin.H{
		"message":   "Profil complété avec succès",
		"companyId": companyID,
		"role":      role,
	})
}

func Me(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur non authentifié"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	id := fmt.Sprintf("%v", userID)

	var user models.User
	col := database.MongoAuthDB.Collection("users")

	err := col.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		if oid, e := primitive.ObjectIDFromHex(id); e == nil {
			err = col.FindOne(ctx, bson.M{"_id": oid}).Decode(&user)
		}
	}

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur introuvable"})
		return
	}

	// ✅ Utilise "userId" au lieu de "user_id" pour cohérence
	c.JSON(http.StatusOK, gin.H{
		"userId":         user.ID, // ✅ Change ici
		"name":           user.Name,
		"email":          user.Email,
		"role":           user.Role,
		"companyId":      user.CompanyID,
		"companyName":    user.CompanyName,
		"isCompanyAdmin": user.IsCompanyAdmin,
		"provider":       user.Provider,
	})
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

	var user models.User
	err := database.MongoAuthDB.Collection("users").
		FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err != nil || user.Email != input.Email {
		c.JSON(http.StatusForbidden, gin.H{"error": "Email non autorisé"})
		return
	}

	update := bson.M{"$set": bson.M{"provider": input.Provider, "provider_id": input.ProviderID}}
	_, err = database.MongoAuthDB.Collection("users").UpdateOne(ctx, bson.M{"_id": userID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur fusion"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Comptes fusionnés avec succès"})
}
