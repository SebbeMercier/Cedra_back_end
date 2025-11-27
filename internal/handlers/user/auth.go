package user

import (
	"cedra_back_end/internal/cache"
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
	"cedra_back_end/internal/utils"
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
	"github.com/minio/minio-go/v7"
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

	// ✅ Paralléliser la vérification email ET le hashing du mot de passe
	type checkResult struct {
		exists bool
		err    error
	}
	checkChan := make(chan checkResult, 1)
	hashChan := make(chan struct {
		hash string
		err  error
	}, 1)

	// Vérifier l'email en parallèle
	go func() {
		var existingUserID gocql.UUID
		err := session.Query("SELECT user_id FROM users_by_email WHERE email = ?", input.Email).Scan(&existingUserID)
		if err == nil {
			// Vérifier aussi le provider
			var provider string
			err = session.Query("SELECT provider FROM users WHERE user_id = ?", existingUserID).Scan(&provider)
			if err == nil && provider == "local" {
				checkChan <- checkResult{exists: true, err: nil}
				return
			}
		}
		checkChan <- checkResult{exists: false, err: nil}
	}()

	// Hasher le mot de passe en parallèle
	go func() {
		hash, err := utils.HashPassword(input.Password)
		hashChan <- struct {
			hash string
			err  error
		}{hash: hash, err: err}
	}()

	// Attendre les résultats
	checkRes := <-checkChan
	if checkRes.exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Un compte avec cet email existe déjà"})
		return
	}

	hashRes := <-hashChan
	if hashRes.err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors du hachage du mot de passe"})
		return
	}
	hashedPassword := hashRes.hash

	// ✅ Crée l'utilisateur avec rôle "pending"
	userID := gocql.TimeUUID()
	userIDStr := userID.String()
	isAdmin := false
	now := time.Now()

	// ✅ Insertions en parallèle avec prepared statements
	errChan := make(chan error, 2)

	// Insert dans users (utiliser prepared statement si disponible)
	go func() {
		stmt := database.GetPreparedInsertUser()
		if stmt != nil {
			errChan <- stmt.Bind(userID, input.Email, hashedPassword, input.Name, "pending", "local", "", nil, "", isAdmin, now, now).Exec()
		} else {
			errChan <- session.Query(`INSERT INTO users (user_id, email, password, name, role, provider, provider_id, company_id, company_name, is_company_admin, created_at, updated_at) 
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				userID, input.Email, hashedPassword, input.Name, "pending", "local", "", nil, "", isAdmin, now, now).Exec()
		}
	}()

	// Insert dans users_by_email (utiliser prepared statement si disponible)
	go func() {
		stmt := database.GetPreparedInsertUserByEmail()
		if stmt != nil {
			errChan <- stmt.Bind(input.Email, userID).Exec()
		} else {
			errChan <- session.Query("INSERT INTO users_by_email (email, user_id) VALUES (?, ?)", input.Email, userID).Exec()
		}
	}()

	// Attendre les 2 insertions
	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			log.Printf("❌ Erreur insertion utilisateur: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur création utilisateur"})
			return
		}
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

	// ✅ Pré-charger le cache utilisateur pour les prochaines requêtes (async)
	go func() {
		ctx := context.Background()
		jsonData, _ := json.Marshal(user)
		database.Redis.Set(ctx, "user:"+userIDStr, jsonData, 5*time.Minute)
	}()

	// Log seulement en mode debug
	if os.Getenv("DEBUG") == "true" {
		log.Printf("✅ Utilisateur créé: %s avec rôle pending", user.Email)
	}

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

	// ✅ Utiliser prepared statement pour améliorer les performances
	var userID gocql.UUID
	stmt := database.GetPreparedGetUserByEmail()
	if stmt != nil {
		err = stmt.Bind(input.Email).Scan(&userID)
	} else {
		// Fallback si prepared statement pas initialisé
		err = session.Query("SELECT user_id FROM users_by_email WHERE email = ?", input.Email).Scan(&userID)
	}
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email ou mot de passe incorrect"})
		return
	}

	// Récupérer seulement les champs nécessaires pour le login
	var (
		email, password, name, role, provider, providerID string
		companyID                                         *gocql.UUID
		companyName                                       string
		isCompanyAdmin                                    bool
	)

	// ✅ Utiliser prepared statement
	stmt2 := database.GetPreparedGetUserByID()
	if stmt2 != nil {
		err = stmt2.Bind(userID).Scan(&email, &password, &name, &role, &provider, &providerID, &companyID, &companyName, &isCompanyAdmin)
	} else {
		err = session.Query(`SELECT email, password, name, role, provider, provider_id, company_id, company_name, is_company_admin 
			FROM users WHERE user_id = ?`, userID).Scan(
			&email, &password, &name, &role, &provider, &providerID, &companyID, &companyName, &isCompanyAdmin)
	}
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email ou mot de passe incorrect"})
		return
	}

	// Vérifier le provider
	if provider != "local" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email ou mot de passe incorrect"})
		return
	}

	// ✅ Vérifier d'abord le cache pour éviter le hashing (gain ~50ms sur logins répétés)
	cached, _ := cache.GetPasswordHashFromCache(input.Email, input.Password)
	if !cached {
		// Si pas en cache, vérifier le mot de passe
		valid, err := utils.VerifyPassword(input.Password, password)
		if err != nil || !valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Email ou mot de passe incorrect"})
			return
		}
		// Mettre en cache pour les prochains logins (15 min)
		cache.SetPasswordHashInCache(input.Email, input.Password)
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

	// ✅ Pré-charger le cache utilisateur pour les prochaines requêtes (async)
	go func() {
		ctx := context.Background()
		jsonData, _ := json.Marshal(user)
		database.Redis.Set(ctx, "user:"+user.ID, jsonData, 5*time.Minute)
	}()

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

	// 1️⃣ Recherche par email (optimisé)
	err = session.Query("SELECT user_id FROM users_by_email WHERE email = ?", email).Scan(&userID)
	if err == nil {
		// Utilisateur existe, récupérer seulement les détails nécessaires
		var (
			emailDB, nameDB, role, providerDB, providerIDDB string
			companyID                                       *gocql.UUID
			companyName                                     string
			isCompanyAdmin                                  bool
		)
		err = session.Query(`SELECT email, name, role, provider, provider_id, company_id, company_name, is_company_admin 
		                     FROM users WHERE user_id = ?`, userID).Scan(
			&emailDB, &nameDB, &role, &providerDB, &providerIDDB, &companyID, &companyName, &isCompanyAdmin)
		if err == nil {
			// Mettre à jour le provider si différent
			if providerDB != provider || providerIDDB != providerID {
				now := time.Now()
				session.Query(`UPDATE users SET provider = ?, provider_id = ?, name = ?, updated_at = ? WHERE user_id = ?`,
					provider, providerID, name, now, userID).Exec()
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
				Role:           role,
				Provider:       provider,
				ProviderID:     providerID,
				CompanyID:      companyIDStr,
				CompanyName:    companyName,
				IsCompanyAdmin: &isCompanyAdmin,
			}
			return user
		}
	}

	// 3️⃣ Création d'un nouvel utilisateur OAuth avec rôle "pending" (parallèle)
	userID = gocql.TimeUUID()
	isAdmin := false
	now := time.Now()

	errChan := make(chan error, 2)

	go func() {
		errChan <- session.Query(`INSERT INTO users (user_id, email, password, name, role, provider, provider_id, company_id, company_name, is_company_admin, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			userID, email, "", name, "pending", provider, providerID, nil, "", isAdmin, now, now).Exec()
	}()

	go func() {
		errChan <- session.Query("INSERT INTO users_by_email (email, user_id) VALUES (?, ?)", email, userID).Exec()
	}()

	// Attendre les 2 insertions
	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			log.Printf("❌ Erreur création utilisateur OAuth: %v", err)
			return models.User{}
		}
	}

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

	claims := jwt.MapClaims{
		"user_id":        user.ID,
		"email":          user.Email,
		"role":           user.Role,
		"isCompanyAdmin": isAdmin,
		"exp":            time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(jwtSecret)

	return tokenString
}

// ================== HANDLERS SUPPLÉMENTAIRES ==================

func CompleteProfile(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur non authentifié"})
		return
	}

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

	// ✅ Crée la company si admin
	var companyID *string
	if input.IsCompanyAdmin && input.CompanyName != "" {
		companyUUID := gocql.TimeUUID()
		companyIDStr := companyUUID.String()
		companyID = &companyIDStr

		// Insérer dans la table companies
		now := time.Now()
		err = session.Query(`INSERT INTO companies (company_id, name, billing_street, billing_postal_code, billing_city, billing_country, created_at) 
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			companyUUID, input.CompanyName, input.BillingStreet, input.BillingPostalCode, input.BillingCity, input.BillingCountry, now).Exec()
		if err != nil {
			log.Printf("❌ Erreur création entreprise: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur création entreprise"})
			return
		}
		log.Printf("✅ Entreprise créée: %s (ID: %s)", input.CompanyName, companyIDStr)

		// ✅ Créer aussi l'adresse de facturation de l'entreprise
		if input.BillingStreet != "" && input.BillingCity != "" {
			addressID := gocql.TimeUUID()
			err = session.Query(`INSERT INTO addresses (address_id, user_id, company_id, street, postal_code, city, country, type, is_default) 
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				addressID, id, companyIDStr, input.BillingStreet, input.BillingPostalCode, input.BillingCity, input.BillingCountry, "billing", true).Exec()
			if err != nil {
				log.Printf("⚠️ Erreur création adresse facturation: %v", err)
			} else {
				log.Printf("✅ Adresse de facturation créée pour %s", input.CompanyName)
			}
		}
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

	// ✅ Invalider puis recharger le cache utilisateur (async)
	cache.InvalidateUserCache(id)
	go func() {
		// Attendre un peu que la DB soit à jour
		time.Sleep(10 * time.Millisecond)
		// Recharger le cache
		cache.GetUserFromCache(id)
	}()

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

	id := fmt.Sprintf("%v", userID)

	// ✅ Utiliser le cache pour améliorer les performances
	user, err := cache.GetUserFromCache(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur introuvable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"userId":         user.ID,
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

func DeleteAccount(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur non authentifié"})
		return
	}

	var input struct {
		Password        string `json:"password"`        // Pour confirmer l'identité (auth locale)
		ConfirmDeletion bool   `json:"confirmDeletion"` // Confirmation explicite
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides"})
		return
	}

	if !input.ConfirmDeletion {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vous devez confirmer la suppression"})
		return
	}

	id := fmt.Sprintf("%v", userID)
	uid, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID utilisateur invalide"})
		return
	}
	userUUID := gocql.UUID(uid)

	// =============================================
	// 1. VÉRIFIER L'IDENTITÉ DE L'UTILISATEUR
	// =============================================

	usersSession, err := database.GetUsersSession()
	if err != nil {
		log.Printf("❌ Erreur session ScyllaDB users: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur connexion base de données"})
		return
	}

	var (
		email, password, provider string
	)

	err = usersSession.Query(`SELECT email, password, provider FROM users WHERE user_id = ?`, userUUID).
		Scan(&email, &password, &provider)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur introuvable"})
		return
	}

	// Vérifier le mot de passe pour les comptes locaux
	if provider == "local" {
		if input.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Mot de passe requis pour confirmer la suppression"})
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(password), []byte(input.Password)) != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Mot de passe incorrect"})
			return
		}
	}

	log.Printf("🗑️ Début de la suppression du compte: %s (%s)", email, id)

	// =============================================
	// 2. SUPPRIMER LES DONNÉES DANS REDIS (PANIER)
	// =============================================

	ctx := context.Background()
	cartKey := "cart:" + id

	err = database.Redis.Del(ctx, cartKey).Err()
	if err != nil {
		log.Printf("⚠️ Erreur suppression panier Redis: %v", err)
	} else {
		log.Printf("✅ Panier supprimé de Redis")
	}

	// Supprimer les sessions et tokens éventuels
	sessionKeys := []string{
		"session:" + id,
		"oauth_redirect:" + id,
		"reset_token:" + email,
	}
	for _, key := range sessionKeys {
		database.Redis.Del(ctx, key)
	}

	// =============================================
	// 3. SUPPRIMER LES ADRESSES (KEYSPACE USERS)
	// =============================================

	// Récupérer toutes les adresses de l'utilisateur
	iter := usersSession.Query(`SELECT address_id FROM addresses WHERE user_id = ?`, id).Iter()
	var addressID gocql.UUID
	addressCount := 0

	for iter.Scan(&addressID) {
		err = usersSession.Query(`DELETE FROM addresses WHERE address_id = ?`, addressID).Exec()
		if err != nil {
			log.Printf("⚠️ Erreur suppression adresse %s: %v", addressID, err)
		} else {
			addressCount++
		}
	}
	iter.Close()
	log.Printf("✅ %d adresse(s) supprimée(s)", addressCount)

	// =============================================
	// 4. SUPPRIMER LES COMMANDES (KEYSPACE ORDERS)
	// =============================================

	ordersSession, err := database.GetOrdersSession()
	if err != nil {
		log.Printf("⚠️ Erreur session ScyllaDB orders: %v", err)
	} else {
		// Supprimer de orders_by_user
		err = ordersSession.Query(`DELETE FROM orders_by_user WHERE user_id = ?`, id).Exec()
		if err != nil {
			log.Printf("⚠️ Erreur suppression orders_by_user: %v", err)
		} else {
			log.Printf("✅ Index orders_by_user supprimé")
		}

		// Récupérer et supprimer toutes les commandes principales
		iter := ordersSession.Query(`SELECT order_id FROM orders WHERE user_id = ?`, id).Iter()
		var orderID gocql.UUID
		orderCount := 0

		for iter.Scan(&orderID) {
			err = ordersSession.Query(`DELETE FROM orders WHERE order_id = ?`, orderID).Exec()
			if err != nil {
				log.Printf("⚠️ Erreur suppression commande %s: %v", orderID, err)
			} else {
				orderCount++
			}
		}
		iter.Close()
		log.Printf("✅ %d commande(s) supprimée(s)", orderCount)
	}

	// =============================================
	// 5. SUPPRIMER LES PRODUITS SI VENDEUR (KEYSPACE PRODUCTS)
	// =============================================

	// Note: Adapter selon votre logique métier
	// Si l'utilisateur a une entreprise, on pourrait supprimer ses produits
	// Mais attention : si plusieurs utilisateurs partagent la même entreprise,
	// il ne faut pas supprimer les produits !
	log.Printf("ℹ️ Suppression des produits non implémentée (nécessite logique métier)")

	// =============================================
	// 6. SUPPRIMER LES IMAGES MINIO
	// =============================================

	// Supprimer les images de profil ou autres fichiers associés
	bucketName := "cedra-images"
	userPrefix := "users/" + id + "/"

	objectsCh := database.MinIO.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Prefix:    userPrefix,
		Recursive: true,
	})

	imageCount := 0
	for object := range objectsCh {
		if object.Err != nil {
			log.Printf("⚠️ Erreur listage MinIO: %v", object.Err)
			continue
		}
		err = database.MinIO.RemoveObject(ctx, bucketName, object.Key, minio.RemoveObjectOptions{})
		if err != nil {
			log.Printf("⚠️ Erreur suppression image %s: %v", object.Key, err)
		} else {
			imageCount++
		}
	}
	log.Printf("✅ %d image(s) supprimée(s) de MinIO", imageCount)

	// =============================================
	// 7. SUPPRIMER DE ELASTICSEARCH
	// =============================================

	// Supprimer l'utilisateur de l'index Elasticsearch si indexé
	// Note: Adapter selon vos index Elasticsearch
	if database.Elastic != nil {
		_, err = database.Elastic.Delete("users", id)
		if err != nil {
			log.Printf("⚠️ Erreur suppression Elasticsearch: %v", err)
		} else {
			log.Printf("✅ Utilisateur supprimé d'Elasticsearch")
		}
	}

	// =============================================
	// 8. SUPPRIMER L'UTILISATEUR (KEYSPACE USERS)
	// =============================================

	// Supprimer de users_by_email (index)
	err = usersSession.Query(`DELETE FROM users_by_email WHERE email = ?`, email).Exec()
	if err != nil {
		log.Printf("⚠️ Erreur suppression users_by_email: %v", err)
	} else {
		log.Printf("✅ Index users_by_email supprimé")
	}

	// Supprimer l'utilisateur principal
	err = usersSession.Query(`DELETE FROM users WHERE user_id = ?`, userUUID).Exec()
	if err != nil {
		log.Printf("❌ Erreur suppression utilisateur: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression du compte"})
		return
	}

	log.Printf("✅ Utilisateur %s (%s) complètement supprimé", email, id)

	c.JSON(http.StatusOK, gin.H{
		"message":    "Votre compte et toutes vos données ont été supprimés définitivement",
		"deleted_at": time.Now().Format(time.RFC3339),
	})
}
