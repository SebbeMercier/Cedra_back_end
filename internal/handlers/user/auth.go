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
	"github.com/gocql/gocql"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/facebook"
	"github.com/markbates/goth/providers/google"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

// ================== AUTH LOCALE ==================

func CreateUser(c *gin.Context) {
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
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

	// Vérifie si l'email existe déjà via users_by_email
	var existingUserID gocql.UUID
	err = session.Query("SELECT user_id FROM users_by_email WHERE email = ?", input.Email).Scan(&existingUserID)
	if err == nil {
		// Vérifier aussi le provider
		var provider string
		err = session.Query("SELECT provider FROM users WHERE user_id = ?", existingUserID).Scan(&provider)
		if err == nil && provider == "local" {
			c.JSON(http.StatusConflict, gin.H{"error": "Un compte avec cet email existe déjà"})
			return
		}
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)

	// ✅ Crée l'utilisateur avec rôle "pending"
	userID := gocql.TimeUUID()
	userIDStr := userID.String()
	isAdmin := false
	now := time.Now()

	// Insert dans users
	err = session.Query(`INSERT INTO users (user_id, email, password, name, role, provider, provider_id, company_id, company_name, is_company_admin, created_at, updated_at) 
	                     VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, input.Email, string(hashedPassword), input.Name, "pending", "local", "", nil, "", isAdmin, now, now).Exec()
	if err != nil {
		log.Printf("❌ Erreur insertion utilisateur: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur création utilisateur"})
		return
	}

	// Insert dans users_by_email pour index
	err = session.Query("INSERT INTO users_by_email (email, user_id) VALUES (?, ?)", input.Email, userID).Exec()
	if err != nil {
		log.Printf("⚠️ Erreur insertion index email: %v", err)
	}

	user := models.User{
		ID:             userIDStr,
		Name:           input.Name,
		Email:          input.Email,
		Password:       string(hashedPassword),
		Role:           "pending",
		Provider:       "local",
		IsCompanyAdmin: &isAdmin,
		CompanyID:      nil,
		CompanyName:    "",
	}

	token := generateJWT(user)

	log.Printf("✅ Utilisateur créé: %s avec rôle pending", user.Email)

	c.JSON(http.StatusCreated, gin.H{
		"token": token,
		"email": user.Email,
		"name":  user.Name,
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email ou mot de passe incorrect"})
		return
	}

	// Récupérer les détails complets
	var (
		email, password, name, role, provider, providerID string
		companyID                                         *gocql.UUID
		companyName                                       string
		isCompanyAdmin                                    bool
		createdAt, updatedAt                              time.Time
	)

	err = session.Query(`SELECT email, password, name, role, provider, provider_id, company_id, company_name, is_company_admin, created_at, updated_at 
	                     FROM users WHERE user_id = ?`, userID).Scan(
		&email, &password, &name, &role, &provider, &providerID, &companyID, &companyName, &isCompanyAdmin, &createdAt, &updatedAt)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email ou mot de passe incorrect"})
		return
	}

	// Vérifier le provider
	if provider != "local" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email ou mot de passe incorrect"})
		return
	}

	// Vérifier le mot de passe
	if bcrypt.CompareHashAndPassword([]byte(password), []byte(input.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email ou mot de passe incorrect"})
		return
	}

	var companyIDStr *string
	if companyID != nil {
		s := companyID.String()
		companyIDStr = &s
	}

	user := models.User{
		ID:             userID.String(),
		Name:           name,
		Email:          email,
		Password:       password,
		Role:           role,
		Provider:       provider,
		ProviderID:     providerID,
		CompanyID:      companyIDStr,
		CompanyName:    companyName,
		IsCompanyAdmin: &isCompanyAdmin,
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
	session, err := database.GetUsersSession()
	if err != nil {
		log.Printf("❌ Erreur session ScyllaDB: %v", err)
		return models.User{}
	}

	var user models.User
	var userID gocql.UUID

	// 1️⃣ Recherche par provider et provider_id (nécessite une table d'index ou scan)
	// Pour simplifier, on cherche d'abord par email
	err = session.Query("SELECT user_id FROM users_by_email WHERE email = ?", email).Scan(&userID)
	if err == nil {
		// Utilisateur existe, récupérer les détails
		var (
			emailDB, password, nameDB, role, providerDB, providerIDDB string
			companyID                                                 *gocql.UUID
			companyName                                               string
			isCompanyAdmin                                            bool
			createdAt, updatedAt                                      time.Time
		)
		err = session.Query(`SELECT email, password, name, role, provider, provider_id, company_id, company_name, is_company_admin, created_at, updated_at 
		                     FROM users WHERE user_id = ?`, userID).Scan(
			&emailDB, &password, &nameDB, &role, &providerDB, &providerIDDB, &companyID, &companyName, &isCompanyAdmin, &createdAt, &updatedAt)
		if err == nil {
			// Mettre à jour le provider si différent
			if providerDB != provider || providerIDDB != providerID {
				now := time.Now()
				session.Query(`UPDATE users SET provider = ?, provider_id = ?, name = ?, updated_at = ? WHERE user_id = ?`,
					provider, providerID, name, now, userID).Exec()
				log.Printf("🔄 Compte existant fusionné avec provider %s : %s", provider, email)
			}
			var companyIDStr *string
			if companyID != nil {
				s := companyID.String()
				companyIDStr = &s
			}
			user = models.User{
				ID:             userID.String(),
				Email:          emailDB,
				Name:           nameDB,
				Password:       password,
				Role:           role,
				Provider:       provider,
				ProviderID:     providerID,
				CompanyID:      companyIDStr,
				CompanyName:    companyName,
				IsCompanyAdmin: &isCompanyAdmin,
			}
			log.Printf("✅ Utilisateur OAuth existant trouvé : %s", email)
			return user
		}
	}

	// 3️⃣ Création d'un nouvel utilisateur OAuth avec rôle "pending"
	userID = gocql.TimeUUID()
	isAdmin := false
	now := time.Now()

	err = session.Query(`INSERT INTO users (user_id, email, password, name, role, provider, provider_id, company_id, company_name, is_company_admin, created_at, updated_at) 
	                     VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, email, "", name, "pending", provider, providerID, nil, "", isAdmin, now, now).Exec()
	if err != nil {
		log.Printf("❌ Erreur création utilisateur OAuth: %v", err)
		return models.User{}
	}

	// Insert dans users_by_email
	session.Query("INSERT INTO users_by_email (email, user_id) VALUES (?, ?)", email, userID).Exec()

	user = models.User{
		ID:             userID.String(),
		Email:          email,
		Name:           name,
		Provider:       provider,
		ProviderID:     providerID,
		Role:           "pending",
		IsCompanyAdmin: &isAdmin,
	}
	log.Printf("🆕 Utilisateur OAuth créé (%s) avec rôle pending : %s", provider, email)
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

	// Crée la company si admin (pour l'instant, on stocke juste l'ID comme string)
	var companyID *string
	if input.IsCompanyAdmin && input.CompanyName != "" {
		// TODO: Créer une table companies dans ScyllaDB si nécessaire
		// Pour l'instant, on génère juste un UUID
		companyUUID := gocql.TimeUUID()
		companyIDStr := companyUUID.String()
		companyID = &companyIDStr
		log.Printf("✅ Company ID généré: %s", companyIDStr)
	}

	// ✅ Définit le rôle selon le type de compte
	role := "customer" // Par défaut : client particulier
	if input.IsCompanyAdmin {
		role = "company-customer" // Client professionnel
	}

	// Met à jour l'utilisateur
	now := time.Now()
	var companyUUID *gocql.UUID
	if companyID != nil {
		cid, err := uuid.Parse(*companyID)
		if err == nil {
			cu := gocql.UUID(cid)
			companyUUID = &cu
		}
	}

	err = session.Query(`UPDATE users SET name = ?, is_company_admin = ?, company_name = ?, role = ?, company_id = ?, updated_at = ? WHERE user_id = ?`,
		input.Name, input.IsCompanyAdmin, input.CompanyName, role, companyUUID, now, userUUID).Exec()
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

	var (
		email, password, name, role, provider, providerID string
		companyID                                         *gocql.UUID
		companyName                                       string
		isCompanyAdmin                                    bool
		createdAt, updatedAt                              time.Time
	)

	err = session.Query(`SELECT email, password, name, role, provider, provider_id, company_id, company_name, is_company_admin, created_at, updated_at 
	                     FROM users WHERE user_id = ?`, userUUID).Scan(
		&email, &password, &name, &role, &provider, &providerID, &companyID, &companyName, &isCompanyAdmin, &createdAt, &updatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur introuvable"})
		return
	}

	var companyIDStr *string
	if companyID != nil {
		s := companyID.String()
		companyIDStr = &s
	}

	c.JSON(http.StatusOK, gin.H{
		"userId":         id,
		"name":           name,
		"email":          email,
		"role":           role,
		"companyId":      companyIDStr,
		"companyName":    companyName,
		"isCompanyAdmin": isCompanyAdmin,
		"provider":       provider,
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

	session, err := database.GetUsersSession()
	if err != nil {
		log.Printf("❌ Erreur session ScyllaDB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID utilisateur invalide"})
		return
	}
	userUUID := gocql.UUID(uid)

	// Vérifier que l'email correspond
	var email string
	err = session.Query("SELECT email FROM users WHERE user_id = ?", userUUID).Scan(&email)
	if err != nil || email != input.Email {
		c.JSON(http.StatusForbidden, gin.H{"error": "Email non autorisé"})
		return
	}

	// Mettre à jour le provider
	now := time.Now()
	err = session.Query(`UPDATE users SET provider = ?, provider_id = ?, updated_at = ? WHERE user_id = ?`,
		input.Provider, input.ProviderID, now, userUUID).Exec()
	if err != nil {
		log.Printf("❌ Erreur mise à jour provider: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur fusion"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Comptes fusionnés avec succès"})
}
