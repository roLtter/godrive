package files

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type patchFileRequest struct {
	Name     *string `json:"name"`
	FolderID *int64  `json:"folder_id"`
}

// Patch handles PATCH /api/files/:id — rename and/or move file to another folder.
func (h *Handler) Patch(c *gin.Context) {
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

	var req patchFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if req.Name == nil && req.FolderID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provide name and/or folder_id"})
		return
	}

	var newName string
	hasName := false
	if req.Name != nil {
		newName = strings.TrimSpace(*req.Name)
		if newName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name must not be empty"})
			return
		}
		hasName = true
	}

	var newFolderID int64
	hasFolder := false
	if req.FolderID != nil {
		if *req.FolderID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid folder_id"})
			return
		}
		exists, err := h.folderBelongsToUser(c, userID, *req.FolderID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify folder"})
			return
		}
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "folder not found"})
			return
		}
		newFolderID = *req.FolderID
		hasFolder = true
	}

	ctx := c.Request.Context()
	var out uploadResponse

	switch {
	case hasName && hasFolder:
		err = h.db.QueryRowContext(ctx, `
			UPDATE files
			SET name = $1, folder_id = $2
			WHERE id = $3 AND user_id = $4
			RETURNING id, folder_id, name, size, mime, s3_key, created_at
		`, newName, newFolderID, fileID, userID).Scan(
			&out.ID, &out.FolderID, &out.Name, &out.Size, &out.Mime, &out.S3Key, &out.CreatedAt,
		)
	case hasName:
		err = h.db.QueryRowContext(ctx, `
			UPDATE files
			SET name = $1
			WHERE id = $2 AND user_id = $3
			RETURNING id, folder_id, name, size, mime, s3_key, created_at
		`, newName, fileID, userID).Scan(
			&out.ID, &out.FolderID, &out.Name, &out.Size, &out.Mime, &out.S3Key, &out.CreatedAt,
		)
	default: // hasFolder only
		err = h.db.QueryRowContext(ctx, `
			UPDATE files
			SET folder_id = $1
			WHERE id = $2 AND user_id = $3
			RETURNING id, folder_id, name, size, mime, s3_key, created_at
		`, newFolderID, fileID, userID).Scan(
			&out.ID, &out.FolderID, &out.Name, &out.Size, &out.Mime, &out.S3Key, &out.CreatedAt,
		)
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update file"})
		return
	}

	c.JSON(http.StatusOK, out)
}
