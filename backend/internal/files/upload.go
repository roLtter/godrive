package files

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
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

// ErrQuotaExceeded is returned when user storage would exceed quota.
var ErrQuotaExceeded = errors.New("storage quota exceeded")

// Handler provides files endpoints.
type Handler struct {
	db           *postgres.Client
	storage      *minioClient.Client
	maxUploadSize int64
	allowedMIME  map[string]struct{}
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
func NewHandler(db *postgres.Client, storage *minioClient.Client, maxUploadSizeBytes int64, allowedMIMEs []string) *Handler {
	allowed := make(map[string]struct{}, len(allowedMIMEs))
	for _, item := range allowedMIMEs {
		value := strings.TrimSpace(strings.ToLower(item))
		if value != "" {
			allowed[value] = struct{}{}
		}
	}
	return &Handler{
		db:            db,
		storage:       storage,
		maxUploadSize: maxUploadSizeBytes,
		allowedMIME:   allowed,
	}
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
	if fileHeader.Size <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty file is not allowed"})
		return
	}
	if fileHeader.Size > h.maxUploadSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file exceeds max upload size"})
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

	detectedMIME, streamReader, err := detectMIMEAndPrepareReader(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to inspect file mime"})
		return
	}
	if _, ok := h.allowedMIME[detectedMIME]; !ok {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "unsupported file mime type"})
		return
	}

	ctx := c.Request.Context()
	out, err := h.reserveUploadInDB(ctx, userID, folderID, fileHeader.Filename, fileHeader.Size, detectedMIME, objectKey)
	if errors.Is(err, ErrQuotaExceeded) {
		c.JSON(http.StatusInsufficientStorage, gin.H{"error": "storage quota exceeded"})
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reserve upload"})
		return
	}

	if err := h.storage.PutObject(ctx, objectKey, streamReader, fileHeader.Size, detectedMIME); err != nil {
		if rbErr := h.releaseUploadReservation(ctx, out.ID, userID, fileHeader.Size); rbErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload file and rollback quota"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload file"})
		return
	}

	c.JSON(http.StatusCreated, out)
}

func (h *Handler) reserveUploadInDB(ctx context.Context, userID string, folderID int64, filename string, size int64, mimeType, objectKey string) (uploadResponse, error) {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return uploadResponse{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var used, quota int64
	err = tx.QueryRowContext(ctx, `
		SELECT storage_used_bytes, storage_quota_bytes
		FROM users
		WHERE id = $1
		FOR UPDATE
	`, userID).Scan(&used, &quota)
	if err != nil {
		return uploadResponse{}, err
	}
	if used+size > quota {
		return uploadResponse{}, ErrQuotaExceeded
	}

	var out uploadResponse
	err = tx.QueryRowContext(ctx, `
		INSERT INTO files (user_id, folder_id, name, size, mime, s3_key)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, folder_id, name, size, mime, s3_key, created_at
	`, userID, folderID, filename, size, mimeType, objectKey).
		Scan(&out.ID, &out.FolderID, &out.Name, &out.Size, &out.Mime, &out.S3Key, &out.CreatedAt)
	if err != nil {
		return uploadResponse{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE users
		SET storage_used_bytes = storage_used_bytes + $1
		WHERE id = $2
	`, size, userID); err != nil {
		return uploadResponse{}, err
	}

	if err := tx.Commit(); err != nil {
		return uploadResponse{}, err
	}
	return out, nil
}

func (h *Handler) releaseUploadReservation(ctx context.Context, fileID int64, userID string, size int64) error {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM files
		WHERE id = $1 AND user_id = $2
	`, fileID, userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE users
		SET storage_used_bytes = GREATEST(0, storage_used_bytes - $1)
		WHERE id = $2
	`, size, userID); err != nil {
		return err
	}
	return tx.Commit()
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

func detectMIMEAndPrepareReader(file io.Reader) (string, io.Reader, error) {
	header := make([]byte, 512)
	n, err := file.Read(header)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", nil, err
	}
	header = header[:n]

	detected := strings.ToLower(strings.TrimSpace(http.DetectContentType(header)))
	if mediaType, _, parseErr := mime.ParseMediaType(detected); parseErr == nil {
		detected = strings.ToLower(mediaType)
	}

	reader := io.MultiReader(bytes.NewReader(header), file)
	return detected, reader, nil
}
