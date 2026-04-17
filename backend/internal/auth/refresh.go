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
)

// RefreshHandler handles POST /refresh requests with token rotation.
type RefreshHandler struct {
	db           *postgres.Client
	tokenIssuer  *TokenIssuer
	refreshStore *RefreshStore
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type refreshResponse struct {
	AccessToken      string    `json:"access_token"`
	TokenType        string    `json:"token_type"`
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshToken     string    `json:"refresh_token"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
}

// NewRefreshHandler creates refresh endpoint handler.
func NewRefreshHandler(db *postgres.Client, tokenIssuer *TokenIssuer, refreshStore *RefreshStore) *RefreshHandler {
	return &RefreshHandler{
		db:           db,
		tokenIssuer:  tokenIssuer,
		refreshStore: refreshStore,
	}
}

// Handle rotates refresh token and returns new token pair.
func (h *RefreshHandler) Handle(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	req.RefreshToken = strings.TrimSpace(req.RefreshToken)
	if req.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token is required"})
		return
	}

	newRefreshToken, refreshExpiresAt, err := h.tokenIssuer.IssueRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}

	userID, err := h.refreshStore.Rotate(c.Request.Context(), req.RefreshToken, newRefreshToken, time.Until(refreshExpiresAt))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	userEmail, err := h.getUserEmailByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process refresh"})
		return
	}

	accessToken, accessExpiresAt, err := h.tokenIssuer.IssueAccessToken(userID, userEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}

	c.JSON(http.StatusOK, refreshResponse{
		AccessToken:      accessToken,
		TokenType:        "Bearer",
		AccessExpiresAt:  accessExpiresAt,
		RefreshToken:     newRefreshToken,
		RefreshExpiresAt: refreshExpiresAt,
	})
}

func (h *RefreshHandler) getUserEmailByID(ctx context.Context, userID string) (string, error) {
	const query = `
		SELECT email
		FROM users
		WHERE id = $1
		LIMIT 1
	`
	var email string
	if err := h.db.QueryRowContext(ctx, query, userID).Scan(&email); err != nil {
		return "", err
	}
	return email, nil
}
