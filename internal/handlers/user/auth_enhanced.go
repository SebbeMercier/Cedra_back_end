package user

import (
"cedra_back_end/internal/cache"
"cedra_back_end/internal/utils"
"log"
"net/http"
"time"

"github.com/gin-gonic/gin"
)

type LoginResponse struct {
AccessToken  string `json:"access_token"`
RefreshToken string `json:"refresh_token"`
ExpiresIn    int64  `json:"expires_in"`
TokenType    string `json:"token_type"`
User         gin.H  `json:"user"`
}

type RefreshTokenRequest struct {
RefreshToken string `json:"refresh_token" binding:"required"`
}

type LogoutRequest struct {
LogoutAll bool `json:"logout_all"`
}

func GenerateAuthTokens(userID, email, role string, isCompanyAdmin bool) (*LoginResponse, error) {
accessToken, tokenID, err := utils.GenerateAccessToken(userID, email, role, isCompanyAdmin)
if err != nil {
return nil, err
}

refreshToken, err := utils.GenerateRefreshToken()
if err != nil {
return nil, err
}

err = cache.StoreRefreshToken(userID, refreshToken, 30*24*time.Hour)
if err != nil {
log.Printf("⚠️ Erreur stockage refresh token: %v", err)
}

log.Printf("✅ Tokens générés - Access: %s, Refresh stocké pour user: %s", tokenID, userID)

return &LoginResponse{
AccessToken:  accessToken,
RefreshToken: refreshToken,
ExpiresIn:    900,
TokenType:    "Bearer",
User: gin.H{
"user_id":          userID,
"email":            email,
"role":             role,
"is_company_admin": isCompanyAdmin,
},
}, nil
}

func RefreshAccessToken(c *gin.Context) {
var req RefreshTokenRequest
if err := c.ShouldBindJSON(&req); err != nil {
c.JSON(http.StatusBadRequest, gin.H{"error": "Refresh token manquant"})
return
}

authHeader := c.GetHeader("Authorization")
if authHeader == "" {
c.JSON(http.StatusUnauthorized, gin.H{"error": "Access token manquant"})
return
}

tokenString := authHeader[7:]
claims, err := utils.ParseAccessToken(tokenString)
if err != nil {
c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalide"})
return
}

userID := claims.UserID

if cache.IsUserBanned(userID) {
c.JSON(http.StatusUnauthorized, gin.H{"error": "Compte banni"})
return
}

storedRefreshToken, err := cache.GetRefreshToken(userID)
if err != nil {
log.Printf("❌ Refresh token non trouvé pour user %s: %v", userID, err)
c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token invalide ou expiré"})
return
}

if storedRefreshToken != req.RefreshToken {
log.Printf("❌ Refresh token ne correspond pas pour user %s", userID)
c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token invalide"})
return
}

newAccessToken, tokenID, err := utils.GenerateAccessToken(
claims.UserID,
claims.Email,
claims.Role,
claims.IsCompanyAdmin,
)
if err != nil {
log.Printf("❌ Erreur génération nouveau token: %v", err)
c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur génération token"})
return
}

log.Printf("✅ Access token renouvelé - TokenID: %s, User: %s", tokenID, userID)

c.JSON(http.StatusOK, gin.H{
"access_token": newAccessToken,
"expires_in":   900,
"token_type":   "Bearer",
})
}

func Logout(c *gin.Context) {
userID, exists := c.Get("user_id")
if !exists {
c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
return
}

tokenID, _ := c.Get("token_id")
userIDStr := userID.(string)
tokenIDStr := tokenID.(string)

var req LogoutRequest
_ = c.ShouldBindJSON(&req)

if req.LogoutAll {
err := cache.DeleteAllRefreshTokens(userIDStr)
if err != nil {
log.Printf("⚠️ Erreur suppression refresh tokens: %v", err)
}
log.Printf("✅ Logout all devices pour user: %s", userIDStr)
} else {
err := cache.DeleteRefreshToken(userIDStr)
if err != nil {
log.Printf("⚠️ Erreur suppression refresh token: %v", err)
}
log.Printf("✅ Logout device pour user: %s", userIDStr)
}

claims, _ := utils.ParseAccessToken(c.GetHeader("Authorization")[7:])
if claims != nil {
duration := utils.GetTokenExpirationDuration(claims)
err := cache.BlacklistToken(tokenIDStr, duration)
if err != nil {
log.Printf("⚠️ Erreur blacklist token: %v", err)
}
log.Printf("✅ Token blacklisté: %s (expire dans %v)", tokenIDStr, duration)
}

c.JSON(http.StatusOK, gin.H{
"message": "Déconnexion réussie",
})
}

func GetActiveSessions(c *gin.Context) {
userID, exists := c.Get("user_id")
if !exists {
c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
return
}

userIDStr := userID.(string)

devices, err := cache.GetAllUserDevices(userIDStr)
if err != nil {
log.Printf("❌ Erreur récupération devices: %v", err)
c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur récupération sessions"})
return
}

c.JSON(http.StatusOK, gin.H{
"sessions": devices,
"count":    len(devices),
})
}

func RevokeSession(c *gin.Context) {
targetUserID := c.Param("user_id")

role, _ := c.Get("role")
if role != "admin" {
c.JSON(http.StatusForbidden, gin.H{"error": "Accès refusé"})
return
}

err := cache.DeleteAllRefreshTokens(targetUserID)
if err != nil {
log.Printf("❌ Erreur révocation sessions: %v", err)
c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur révocation"})
return
}

log.Printf("✅ Sessions révoquées pour user: %s", targetUserID)

c.JSON(http.StatusOK, gin.H{
"message": "Sessions révoquées",
})
}

func BanUserAccount(c *gin.Context) {
targetUserID := c.Param("user_id")

role, _ := c.Get("role")
if role != "admin" {
c.JSON(http.StatusForbidden, gin.H{"error": "Accès refusé"})
return
}

err := cache.BanUser(targetUserID)
if err != nil {
log.Printf("❌ Erreur ban user: %v", err)
c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur ban"})
return
}

_ = cache.DeleteAllRefreshTokens(targetUserID)

log.Printf("✅ User banni: %s", targetUserID)

c.JSON(http.StatusOK, gin.H{
"message": "Utilisateur banni",
})
}

func UnbanUserAccount(c *gin.Context) {
targetUserID := c.Param("user_id")

role, _ := c.Get("role")
if role != "admin" {
c.JSON(http.StatusForbidden, gin.H{"error": "Accès refusé"})
return
}

err := cache.UnbanUser(targetUserID)
if err != nil {
log.Printf("❌ Erreur unban user: %v", err)
c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur unban"})
return
}

log.Printf("✅ User débanni: %s", targetUserID)

c.JSON(http.StatusOK, gin.H{
"message": "Utilisateur débanni",
})
}
