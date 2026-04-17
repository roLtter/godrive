package auth

import (
	"cloudstore/backend/internal/db/postgres"
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost      = 12
	minPasswordSize = 8
)

// RegisterHandler handles POST /register requests.
type RegisterHandler struct {
	db *postgres.Client
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// NewRegisterHandler creates register endpoint handler.
func NewRegisterHandler(db *postgres.Client) *RegisterHandler {
	return &RegisterHandler{db: db}
}

// Handle creates a new user with bcrypt password hash.
func (h *RegisterHandler) Handle(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || !strings.Contains(email, "@") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return
	}
	if len(req.Password) < minPasswordSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 8 characters"})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	resp, err := h.createUser(c.Request.Context(), email, string(passwordHash))
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "user already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *RegisterHandler) createUser(ctx context.Context, email, passwordHash string) (registerResponse, error) {
	const query = `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, created_at
	`

	var out registerResponse
	err := h.db.QueryRowContext(ctx, query, email, passwordHash).Scan(&out.ID, &out.Email, &out.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return registerResponse{}, errors.New("user was not created")
		}
		return registerResponse{}, err
	}
	return out, nil
}
