package folders

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// folderDetailFile is a file row shown inside a folder view (non-deleted only).
type folderDetailFile struct {
	ID        int64     `json:"id"`
	FolderID  int64     `json:"folder_id"`
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	Mime      string    `json:"mime"`
	CreatedAt time.Time `json:"created_at"`
}

// GetByID handles GET /api/folders/:id — folder metadata, breadcrumbs, child folders, and files.
func (h *Handler) GetByID(c *gin.Context) {
	userID, ok := authUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	folderID, ok := folderIDParam(c)
	if !ok {
		return
	}

	ctx := c.Request.Context()
	folder, err := h.getFolderByID(c, userID, folderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "folder not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load folder"})
		return
	}

	breadcrumbs, err := h.loadBreadcrumbs(c, userID, folderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load breadcrumbs"})
		return
	}

	childFolders := make([]folderResponse, 0)
	frows, err := h.db.QueryContext(ctx, `
		SELECT id, user_id, parent_id, name
		FROM folders
		WHERE user_id = $1 AND parent_id = $2
		ORDER BY name ASC
	`, userID, folderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list child folders"})
		return
	}
	defer frows.Close()
	for frows.Next() {
		var f folderResponse
		if err := frows.Scan(&f.ID, &f.UserID, &f.ParentID, &f.Name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan child folders"})
			return
		}
		childFolders = append(childFolders, f)
	}
	if err := frows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list child folders"})
		return
	}

	files := make([]folderDetailFile, 0)
	fileRows, err := h.db.QueryContext(ctx, `
		SELECT id, folder_id, name, size, mime, created_at
		FROM files
		WHERE user_id = $1 AND folder_id = $2 AND deleted_at IS NULL
		ORDER BY name ASC
	`, userID, folderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list files in folder"})
		return
	}
	defer fileRows.Close()
	for fileRows.Next() {
		var f folderDetailFile
		if err := fileRows.Scan(&f.ID, &f.FolderID, &f.Name, &f.Size, &f.Mime, &f.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan files"})
			return
		}
		files = append(files, f)
	}
	if err := fileRows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list files in folder"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"folder":      folder,
		"breadcrumbs": breadcrumbs,
		"folders":     childFolders,
		"files":       files,
	})
}
