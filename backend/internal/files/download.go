package files

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// DownloadByID handles GET /api/files/:id/download — redirects to a presigned MinIO GET URL.
func (h *Handler) DownloadByID(c *gin.Context) {
	userID, ok := authUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	raw := strings.TrimSpace(c.Param("id"))
	fileID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || fileID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file id"})
		return
	}

	h.redirectPresignedDownload(c, userID, fileID)
}

func (h *Handler) redirectPresignedDownload(c *gin.Context, userID string, fileID int64) {
	const query = `
		SELECT s3_key
		FROM files
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
		LIMIT 1
	`
	var s3Key string
	err := h.db.QueryRowContext(c.Request.Context(), query, fileID, userID).Scan(&s3Key)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve file"})
		return
	}

	presigned, err := h.storage.PresignedGetURL(c.Request.Context(), s3Key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create download URL"})
		return
	}

	c.Redirect(http.StatusFound, presigned.String())
}
