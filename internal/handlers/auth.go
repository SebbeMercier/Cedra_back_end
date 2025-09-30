package handlers

import (
	"context"
	"net/http"
	"os"
	"time"

	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/markbates/goth/gothic"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

type ctxKey string

const providerKey ctxKey = "provider"

// ================== AUTH LOCALE ==================

func CreateUser(c *gin.Context) {
    var input struct {
        Name           string `json:"name"`
        Email          string `json:"email"`
        Password       string `json:"password"`
        IsCompanyAdmin bool   `json:"isCompanyAdmin"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // ⚡ Vérifier si l'email existe déjà
    var existing models.User
    err := database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
        "email": input.Email,
        "provider": "local",
    }).Decode(&existing)

    if err == nil {
    // utilisateur trouvé → doublon
    c.JSON(http.StatusConflict, gin.H{
        "error": "Un compte avec cet email existe déjà",
        "email": input.Email,
    })
    return
}

    // ✅ hash password
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur hash mot de passe"})
        return
    }

    // Toujours customer → isCompanyAdmin est juste un flag
    newUser := models.User{
        ID:             primitive.NewObjectID(),
        Name:           input.Name,
        Email:          input.Email,
        Password:       string(hashedPassword),
        Role:           "customer",
        IsCompanyAdmin: input.IsCompanyAdmin,
        Provider:       "local",
        CreatedAt:      primitive.NewDateTimeFromTime(time.Now()),
    }

    _, err = database.MongoAuthDB.Collection("users").InsertOne(ctx, newUser)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, gin.H{
        "id":             newUser.ID.Hex(),
        "email":          newUser.Email,
        "role":           newUser.Role,
        "isCompanyAdmin": newUser.IsCompanyAdmin,
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

	token := generateJWT(user)
	c.JSON(http.StatusOK, gin.H{
		"token":          token,
		"userId":         user.ID.Hex(),
		"name":           user.Name, // ✅ renvoie aussi le nom
		"email":          user.Email,
		"role":           user.Role,
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

	c.Request = c.Request.WithContext(
		context.WithValue(c.Request.Context(), providerKey, provider),
	)

	gothic.BeginAuthHandler(c.Writer, c.Request)
}

func CallbackAuth(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "aucun provider spécifié"})
		return
	}

	c.Request = c.Request.WithContext(
		context.WithValue(c.Request.Context(), providerKey, provider),
	)

	userInfo, err := gothic.CompleteUserAuth(c.Writer, c.Request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err = database.MongoAuthDB.Collection("users").FindOne(ctx, bson.M{
		"provider":    provider,
		"provider_id": userInfo.UserID,
	}).Decode(&user)

	if err != nil {
		// Création d’un nouvel utilisateur social
		user = models.User{
			ID:         primitive.NewObjectID(),
			Email:      userInfo.Email,
			Name:       userInfo.Name, // ✅ nom récupéré du provider si dispo
			Provider:   provider,
			ProviderID: userInfo.UserID,
			Role:       "customer",
			CreatedAt:  primitive.NewDateTimeFromTime(time.Now()),
		}
		_, err := database.MongoAuthDB.Collection("users").InsertOne(ctx, user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur enregistrement utilisateur"})
			return
		}
	}

	token := generateJWT(user)
	c.JSON(http.StatusOK, gin.H{
		"token":    token,
		"provider": provider,
		"email":    user.Email,
		"name":     user.Name, // ✅ renvoie aussi le nom
		"role":     user.Role,
	})
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
		"name":           name, // ✅ renvoie le nom dans /me
		"isCompanyAdmin": isCompanyAdmin,
	})
}

