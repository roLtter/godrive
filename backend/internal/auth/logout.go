package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// LogoutHandler handles POST /logout requests.
type LogoutHandler struct {
	refreshStore *RefreshStore
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// NewLogoutHandler creates logout endpoint handler.
func NewLogoutHandler(refreshStore *RefreshStore) *LogoutHandler {
	return &LogoutHandler{refreshStore: refreshStore}
}

// Handle invalidates refresh token in Redis.
func (h *LogoutHandler) Handle(c *gin.Context) {
	var req logoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	req.RefreshToken = strings.TrimSpace(req.RefreshToken)
	if req.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token is required"})
		return
	}

	if err := h.refreshStore.Invalidate(c.Request.Context(), req.RefreshToken); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}
