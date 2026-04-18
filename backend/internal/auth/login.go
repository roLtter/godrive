package auth

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"cloudstore/backend/internal/db/postgres"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// LoginHandler handles POST /login requests.
type LoginHandler struct {
	db          *postgres.Client
	tokenIssuer *TokenIssuer
	refreshStore *RefreshStore
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken      string    `json:"access_token"`
	TokenType        string    `json:"token_type"`
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshToken     string    `json:"refresh_token"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
}

type userCredentials struct {
	ID           string
	Email        string
	PasswordHash string
}

// NewLoginHandler creates login endpoint handler.
func NewLoginHandler(db *postgres.Client, tokenIssuer *TokenIssuer, refreshStore *RefreshStore) *LoginHandler {
	return &LoginHandler{
		db:          db,
		tokenIssuer: tokenIssuer,
		refreshStore: refreshStore,
	}
}

// Handle validates credentials and returns JWT token.
func (h *LoginHandler) Handle(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and password are required"})
		return
	}

	user, err := h.getUserByEmail(c.Request.Context(), email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process login"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	accessToken, expiresAt, err := h.tokenIssuer.IssueAccessToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}
	refreshToken, refreshExpiresAt, err := h.tokenIssuer.IssueRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}
	if err := h.refreshStore.Save(c.Request.Context(), refreshToken, user.ID, time.Until(refreshExpiresAt)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist refresh token"})
		return
	}

	c.JSON(http.StatusOK, loginResponse{
		AccessToken:      accessToken,
		TokenType:        "Bearer",
		AccessExpiresAt:  expiresAt,
		RefreshToken:     refreshToken,
		RefreshExpiresAt: refreshExpiresAt,
	})
}

func (h *LoginHandler) getUserByEmail(ctx context.Context, email string) (userCredentials, error) {
	const query = `
		SELECT id, email, password_hash
		FROM users
		WHERE email = $1
		LIMIT 1
	`

	var user userCredentials
	err := h.db.QueryRowContext(ctx, query, email).Scan(&user.ID, &user.Email, &user.PasswordHash)
	if err != nil {
		return userCredentials{}, err
	}
	return user, nil
}
