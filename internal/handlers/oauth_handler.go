package handlers

import (
    "context"
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/markbates/goth/gothic"
)

type ctxKey string

const ProviderKey ctxKey = "provider"

func BeginAuth(c *gin.Context) {
    provider := c.Param("provider")
    if provider == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "aucun provider spécifié"})
        return
    }

    c.Request = c.Request.WithContext(
        context.WithValue(c.Request.Context(), ProviderKey, provider),
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
        context.WithValue(c.Request.Context(), ProviderKey, provider),
    )

    user, err := gothic.CompleteUserAuth(c.Writer, c.Request)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "provider": user.Provider,
        "email":    user.Email,
        "user_id":  user.UserID,
    })
}