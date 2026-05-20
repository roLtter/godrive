package files

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// SoftDelete handles DELETE /api/files/:id — sets deleted_at (soft delete).
func (h *Handler) SoftDelete(c *gin.Context) {
	userID, ok := authUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	rawID := strings.TrimSpace(c.Param("id"))
	fileID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || fileID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file id"})
		return
	}

	const query = `
		UPDATE files
		SET deleted_at = now()
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`
	res, err := h.db.ExecContext(c.Request.Context(), query, fileID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete file"})
		return
	}
	n, err := res.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete file"})
		return
	}
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	c.Status(http.StatusNoContent)
}
