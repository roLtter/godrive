package files

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cloudstore/backend/internal/db/postgres"
	"cloudstore/backend/internal/middleware"
	minioClient "cloudstore/backend/internal/storage/minio"
	"github.com/gin-gonic/gin"
)

// Handler provides files endpoints.
type Handler struct {
	db      *postgres.Client
	storage *minioClient.Client
}

type uploadResponse struct {
	ID        int64     `json:"id"`
	FolderID  int64     `json:"folder_id"`
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	Mime      string    `json:"mime"`
	S3Key     string    `json:"s3_key"`
	CreatedAt time.Time `json:"created_at"`
}

// NewHandler creates files handler.
func NewHandler(db *postgres.Client, storage *minioClient.Client) *Handler {
	return &Handler{db: db, storage: storage}
}

// Upload handles POST /api/upload multipart file upload stream to MinIO.
func (h *Handler) Upload(c *gin.Context) {
	userID, ok := authUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	folderID, err := strconv.ParseInt(strings.TrimSpace(c.PostForm("folder_id")), 10, 64)
	if err != nil || folderID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid folder_id"})
		return
	}

	exists, err := h.folderBelongsToUser(c, userID, folderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify folder"})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "folder not found"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open uploaded file"})
		return
	}
	defer file.Close()

	objectKey, err := buildObjectKey(userID, fileHeader.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to prepare upload key"})
		return
	}

	contentType := strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if err := h.storage.PutObject(c.Request.Context(), objectKey, file, fileHeader.Size, contentType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload file"})
		return
	}

	const query = `
		INSERT INTO files (user_id, folder_id, name, size, mime, s3_key)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, folder_id, name, size, mime, s3_key, created_at
	`
	var out uploadResponse
	if err := h.db.QueryRowContext(c.Request.Context(), query,
		userID, folderID, fileHeader.Filename, fileHeader.Size, contentType, objectKey).
		Scan(&out.ID, &out.FolderID, &out.Name, &out.Size, &out.Mime, &out.S3Key, &out.CreatedAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist file metadata"})
		return
	}

	c.JSON(http.StatusCreated, out)
}

func (h *Handler) folderBelongsToUser(c *gin.Context, userID string, folderID int64) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM folders WHERE id = $1 AND user_id = $2
		)
	`
	var exists bool
	if err := h.db.QueryRowContext(c.Request.Context(), query, folderID, userID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
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

func buildObjectKey(userID, filename string) (string, error) {
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		ext = ".bin"
	}
	trimmedUser := strings.TrimSpace(userID)
	if trimmedUser == "" {
		return "", errors.New("empty user id")
	}
	return fmt.Sprintf("%s/%d-%s%s", trimmedUser, time.Now().UTC().UnixNano(), hex.EncodeToString(random), ext), nil
}
