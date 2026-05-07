package files

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Download handles GET /api/download?file_id= — redirects to a presigned MinIO GET URL.
func (h *Handler) Download(c *gin.Context) {
	userID, ok := authUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	raw := strings.TrimSpace(c.Query("file_id"))
	if raw == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file_id is required"})
		return
	}
	fileID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || fileID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file_id"})
		return
	}

	const query = `
		SELECT s3_key
		FROM files
		WHERE id = $1 AND user_id = $2
		LIMIT 1
	`
	var s3Key string
	err = h.db.QueryRowContext(c.Request.Context(), query, fileID, userID).Scan(&s3Key)
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
