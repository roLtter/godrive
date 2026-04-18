package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	// ContextUserIDKey stores authenticated user ID in Gin context.
	ContextUserIDKey = "auth_user_id"
	// ContextUserEmailKey stores authenticated user email in Gin context.
	ContextUserEmailKey = "auth_user_email"
)

// JWTAuth validates Bearer JWT token and stores claims in request context.
func JWTAuth(secret string) gin.HandlerFunc {
	secretKey := []byte(secret)

	return func(c *gin.Context) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			c.Abort()
			return
		}

		tokenString := strings.TrimSpace(parts[1])
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			c.Abort()
			return
		}

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
			return secretKey, nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid access token"})
			c.Abort()
			return
		}

		sub, ok := claims["sub"].(string)
		if !ok || strings.TrimSpace(sub) == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid access token"})
			c.Abort()
			return
		}
		email, _ := claims["email"].(string)

		c.Set(ContextUserIDKey, sub)
		c.Set(ContextUserEmailKey, email)
		c.Next()
	}
}
