package files

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	defaultListPage    = 1
	defaultListPerPage = 20
	maxListPerPage     = 100
)

type fileListItem struct {
	ID        int64     `json:"id"`
	FolderID  int64     `json:"folder_id"`
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	Mime      string    `json:"mime"`
	CreatedAt time.Time `json:"created_at"`
}

type listFilesResponse struct {
	Items      []fileListItem `json:"items"`
	Page       int            `json:"page"`
	PerPage    int            `json:"per_page"`
	Total      int64          `json:"total"`
	TotalPages int            `json:"total_pages"`
}

// List handles GET /api/files — paginated file list with optional folder filter and sorting.
func (h *Handler) List(c *gin.Context) {
	userID, ok := authUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	page := parsePositiveInt(c.DefaultQuery("page", strconv.Itoa(defaultListPage)), defaultListPage)
	perPage := parsePositiveInt(c.DefaultQuery("per_page", strconv.Itoa(defaultListPerPage)), defaultListPerPage)
	if perPage > maxListPerPage {
		perPage = maxListPerPage
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * perPage

	sortBy := strings.ToLower(strings.TrimSpace(c.DefaultQuery("sort_by", "created_at")))
	sortOrder := strings.ToLower(strings.TrimSpace(c.DefaultQuery("sort_order", "desc")))
	orderCol, ok := mapSortBy(sortBy)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sort_by (use name, size, created_at)"})
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
	countQuery := `SELECT COUNT(*) FROM files WHERE user_id = $1 AND deleted_at IS NULL`
	countArgs := []any{userID}
	if folderID != nil {
		countQuery += ` AND folder_id = $2`
		countArgs = append(countArgs, *folderID)
	}
	if err := h.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count files"})
		return
	}

	listQuery := fmt.Sprintf(`
		SELECT id, folder_id, name, size, mime, created_at
		FROM files
		WHERE user_id = $1 AND deleted_at IS NULL
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list files"})
		return
	}
	defer rows.Close()

	items := make([]fileListItem, 0)
	for rows.Next() {
		var it fileListItem
		if err := rows.Scan(&it.ID, &it.FolderID, &it.Name, &it.Size, &it.Mime, &it.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan files"})
			return
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list files"})
		return
	}

	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	if total == 0 {
		totalPages = 0
	}

	c.JSON(http.StatusOK, listFilesResponse{
		Items:      items,
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	})
}

func mapSortBy(s string) (column string, ok bool) {
	switch s {
	case "name":
		return "name", true
	case "size":
		return "size", true
	case "created_at":
		return "created_at", true
	default:
		return "", false
	}
}

func mapSortOrder(s string) (dir string, ok bool) {
	switch s {
	case "asc":
		return "ASC", true
	case "desc":
		return "DESC", true
	default:
		return "", false
	}
}

func parsePositiveInt(raw string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 1 {
		return fallback
	}
	return n
}
