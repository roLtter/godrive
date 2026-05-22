package files

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type fileTrashItem struct {
	ID         int64     `json:"id"`
	FolderID   int64     `json:"folder_id"`
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	Mime       string    `json:"mime"`
	CreatedAt  time.Time `json:"created_at"`
	DeletedAt  time.Time `json:"deleted_at"`
}

type listTrashResponse struct {
	Items      []fileTrashItem `json:"items"`
	Page       int             `json:"page"`
	Limit      int             `json:"limit"`
	Total      int64           `json:"total"`
	TotalPages int             `json:"total_pages"`
}

// ListTrash handles GET /api/files/trash — paginated list of soft-deleted files.
func (h *Handler) ListTrash(c *gin.Context) {
	userID, ok := authUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	page, perPage := listPageSize(c)
	offset := (page - 1) * perPage

	sortBy := strings.ToLower(strings.TrimSpace(c.DefaultQuery("sort_by", "deleted_at")))
	sortOrder := strings.ToLower(strings.TrimSpace(c.DefaultQuery("sort_order", "desc")))
	orderCol, ok := mapTrashSortBy(sortBy)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sort_by (use name, size, created_at, deleted_at)"})
		return
	}
	orderDir, ok := mapSortOrder(sortOrder)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sort_order (use asc, desc)"})
		return
	}

	ctx := c.Request.Context()
	folderRaw := strings.TrimSpace(c.Query("folder_id"))
	var folderID *int64
	if folderRaw != "" {
		fid, err := strconv.ParseInt(folderRaw, 10, 64)
		if err != nil || fid <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid folder_id"})
			return
		}
		exists, err := h.folderBelongsToUser(c, userID, fid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify folder"})
			return
		}
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "folder not found"})
			return
		}
		folderID = &fid
	}

	var total int64
	countQuery := `SELECT COUNT(*) FROM files WHERE user_id = $1 AND deleted_at IS NOT NULL`
	countArgs := []any{userID}
	if folderID != nil {
		countQuery += ` AND folder_id = $2`
		countArgs = append(countArgs, *folderID)
	}
	if err := h.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count trashed files"})
		return
	}

	listQuery := fmt.Sprintf(`
		SELECT id, folder_id, name, size, mime, created_at, deleted_at
		FROM files
		WHERE user_id = $1 AND deleted_at IS NOT NULL
	`)
	listArgs := []any{userID}
	argPos := 2
	if folderID != nil {
		listQuery += fmt.Sprintf(` AND folder_id = $%d`, argPos)
		listArgs = append(listArgs, *folderID)
		argPos++
	}
	listQuery += fmt.Sprintf(` ORDER BY %s %s`, orderCol, orderDir)
	listQuery += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, argPos, argPos+1)
	listArgs = append(listArgs, perPage, offset)

	rows, err := h.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list trashed files"})
		return
	}
	defer rows.Close()

	items := make([]fileTrashItem, 0)
	for rows.Next() {
		var it fileTrashItem
		if err := rows.Scan(&it.ID, &it.FolderID, &it.Name, &it.Size, &it.Mime, &it.CreatedAt, &it.DeletedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan trashed files"})
			return
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list trashed files"})
		return
	}

	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	if total == 0 {
		totalPages = 0
	}

	c.JSON(http.StatusOK, listTrashResponse{
		Items:      items,
		Page:       page,
		Limit:      perPage,
		Total:      total,
		TotalPages: totalPages,
	})
}

func mapTrashSortBy(s string) (column string, ok bool) {
	switch s {
	case "name":
		return "name", true
	case "size":
		return "size", true
	case "created_at":
		return "created_at", true
	case "deleted_at":
		return "deleted_at", true
	default:
		return "", false
	}
}
