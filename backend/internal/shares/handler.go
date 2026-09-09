package shares

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloudstore/backend/internal/db/postgres"
	"cloudstore/backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

const defaultShareTTL = 30 * 24 * time.Hour
const maxShareTTL = 365 * 24 * time.Hour

// Handler provides share endpoints.
type Handler struct {
	db      *postgres.Client
	storage ObjectStorage
}

type createShareRequest struct {
	FileID     int64   `json:"file_id"`
	TTLSeconds *int64  `json:"ttl_seconds,omitempty"`
	ExpiresIn  *int64  `json:"expires_in,omitempty"`
	Password   *string `json:"password,omitempty"`
}

type createShareResponse struct {
	ID        int64     `json:"id"`
	FileID    int64     `json:"file_id"`
	Token     string    `json:"token"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

type shareListItem struct {
	ID             int64      `json:"id"`
	FileID         int64      `json:"file_id"`
	Token          string     `json:"token"`
	URL            string     `json:"url"`
	ExpiresAt      time.Time  `json:"expires_at"`
	DownloadCount  int64      `json:"download_count"`
	LastAccessedAt *time.Time `json:"last_accessed_at,omitempty"`
	CreatedAt      *time.Time `json:"created_at,omitempty"`
}

// NewHandler creates shares handler.
func NewHandler(db *postgres.Client, storage ObjectStorage) *Handler {
	return &Handler{db: db, storage: storage}
}

// Create handles POST /api/shares and creates a public share token for a file.
func (h *Handler) Create(c *gin.Context) {
	userID, ok := authUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req createShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if req.FileID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file_id"})
		return
	}

	effectiveTTL := req.TTLSeconds
	if effectiveTTL == nil {
		effectiveTTL = req.ExpiresIn
	}

	expiresAt, err := shareExpiryFromRequest(effectiveTTL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	passwordHash, err := hashSharePassword(req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := newUUIDv4()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate share token"})
		return
	}

	const query = `
		INSERT INTO shares (file_id, token, expires_at, password_hash)
		SELECT f.id, $2, $3, $4
		FROM files f
		WHERE f.id = $1
		  AND f.user_id = $5
		  AND f.deleted_at IS NULL
		RETURNING id, file_id, token, expires_at
	`
	var out createShareResponse
	err = h.db.QueryRowContext(c.Request.Context(), query, req.FileID, token, expiresAt, passwordHash, userID).
		Scan(&out.ID, &out.FileID, &out.Token, &out.ExpiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create share"})
		return
	}

	out.URL = publicShareURL(c, out.Token)
	c.JSON(http.StatusCreated, out)
}

// ListActive handles GET /api/shares and returns active (not expired) shares for current user.
func (h *Handler) ListActive(c *gin.Context) {
	userID, ok := authUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	const query = `
		SELECT s.id, s.file_id, s.token, s.expires_at, s.download_count, s.last_accessed_at
		FROM shares s
		INNER JOIN files f ON f.id = s.file_id
		WHERE f.user_id = $1
		  AND f.deleted_at IS NULL
		  AND s.expires_at > NOW()
		ORDER BY s.expires_at ASC
	`
	rows, err := h.db.QueryContext(c.Request.Context(), query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list shares"})
		return
	}
	defer rows.Close()

	items := make([]shareListItem, 0)
	for rows.Next() {
		var (
			item     shareListItem
			lastSeen sql.NullTime
		)
		if err := rows.Scan(&item.ID, &item.FileID, &item.Token, &item.ExpiresAt, &item.DownloadCount, &lastSeen); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan shares"})
			return
		}
		if lastSeen.Valid {
			ts := lastSeen.Time
			item.LastAccessedAt = &ts
		}
		item.URL = publicShareURL(c, item.Token)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list shares"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

// Revoke handles DELETE /api/shares/:id and revokes a share owned by current user.
func (h *Handler) Revoke(c *gin.Context) {
	userID, ok := authUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	shareID, err := parsePositiveID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid share id"})
		return
	}

	const query = `
		DELETE FROM shares s
		USING files f
		WHERE s.id = $1
		  AND f.id = s.file_id
		  AND f.user_id = $2
	`
	result, err := h.db.ExecContext(c.Request.Context(), query, shareID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke share"})
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "share not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

// Resolve handles GET /s/:token and redirects to a presigned URL.
func (h *Handler) Resolve(c *gin.Context) {
	if h.storage == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage is not configured"})
		return
	}

	token := strings.TrimSpace(c.Param("token"))
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid share token"})
		return
	}

	var (
		s3Key        string
		expiresAt    time.Time
		passwordHash sql.NullString
	)
	const query = `
		SELECT f.s3_key, s.expires_at, s.password_hash
		FROM shares s
		INNER JOIN files f ON f.id = s.file_id
		WHERE s.token = $1
		  AND f.deleted_at IS NULL
		LIMIT 1
	`
	err := h.db.QueryRowContext(c.Request.Context(), query, token).Scan(&s3Key, &expiresAt, &passwordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "share not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve share"})
		return
	}
	if time.Now().UTC().After(expiresAt) {
		c.JSON(http.StatusGone, gin.H{"error": "share expired"})
		return
	}
	if passwordHash.Valid {
		password := strings.TrimSpace(c.Query("password"))
		if password == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "password required"})
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(passwordHash.String), []byte(password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password"})
			return
		}
	}

	presigned, err := h.storage.PresignedGetURL(c.Request.Context(), s3Key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create download URL"})
		return
	}
	if err := h.trackShareAccess(c, token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update share access"})
		return
	}
	c.Redirect(http.StatusFound, presigned.String())
}

func (h *Handler) trackShareAccess(c *gin.Context, token string) error {
	const query = `
		UPDATE shares
		SET download_count = download_count + 1,
		    last_accessed_at = NOW()
		WHERE token = $1
	`
	_, err := h.db.ExecContext(c.Request.Context(), query, token)
	return err
}

func authUserID(c *gin.Context) (string, bool) {
	value, exists := c.Get(middleware.ContextUserIDKey)
	if !exists {
		return "", false
	}
	userID, ok := value.(string)
	if !ok || strings.TrimSpace(userID) == "" {
		return "", false
	}
	return userID, true
}

func publicShareURL(c *gin.Context, token string) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")); forwarded != "" {
		scheme = forwarded
	}
	host := strings.TrimSpace(c.Request.Host)
	if host == "" {
		return "/s/" + token
	}
	return scheme + "://" + host + "/s/" + token
}

func newUUIDv4() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	dst := make([]byte, 36)
	hex.Encode(dst[0:8], b[0:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], b[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], b[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], b[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:36], b[10:16])
	return string(dst), nil
}

func shareExpiryFromRequest(ttlSeconds *int64) (time.Time, error) {
	if ttlSeconds == nil {
		return time.Now().UTC().Add(defaultShareTTL), nil
	}
	if *ttlSeconds <= 0 {
		return time.Time{}, errors.New("ttl_seconds must be greater than zero")
	}
	ttl := time.Duration(*ttlSeconds) * time.Second
	if ttl > maxShareTTL {
		return time.Time{}, errors.New("ttl_seconds exceeds maximum allowed value")
	}
	return time.Now().UTC().Add(ttl), nil
}

func hashSharePassword(password *string) (*string, error) {
	if password == nil {
		return nil, nil
	}
	plain := strings.TrimSpace(*password)
	if plain == "" {
		return nil, errors.New("password must not be empty")
	}
	if len(plain) < 4 {
		return nil, errors.New("password must be at least 4 characters")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	out := string(hashed)
	return &out, nil
}

func parsePositiveID(raw string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}
