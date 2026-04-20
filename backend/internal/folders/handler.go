package folders

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloudstore/backend/internal/db/postgres"
	"cloudstore/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// Handler provides folders CRUD endpoints.
type Handler struct {
	db *postgres.Client
}

type createFolderRequest struct {
	Name     string `json:"name"`
	ParentID *int64 `json:"parent_id"`
}

type renameFolderRequest struct {
	Name string `json:"name"`
}

type folderResponse struct {
	ID       int64      `json:"id"`
	UserID   string     `json:"user_id"`
	ParentID *int64     `json:"parent_id,omitempty"`
	Name     string     `json:"name"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

// NewHandler creates folders handler.
func NewHandler(db *postgres.Client) *Handler {
	return &Handler{db: db}
}

// Create handles POST /api/folders.
func (h *Handler) Create(c *gin.Context) {
	userID, ok := authUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req createFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "folder name is required"})
		return
	}

	if req.ParentID != nil {
		exists, err := h.folderBelongsToUser(c, userID, *req.ParentID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify parent folder"})
			return
		}
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "parent folder not found"})
			return
		}
	}

	const query = `
		INSERT INTO folders (user_id, parent_id, name)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, parent_id, name
	`
	var out folderResponse
	if err := h.db.QueryRowContext(c.Request.Context(), query, userID, req.ParentID, name).
		Scan(&out.ID, &out.UserID, &out.ParentID, &out.Name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create folder"})
		return
	}

	c.JSON(http.StatusCreated, out)
}

// List handles GET /api/folders?parent_id=<id>.
func (h *Handler) List(c *gin.Context) {
	userID, ok := authUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	parentParam := strings.TrimSpace(c.Query("parent_id"))
	var (
		rows *sql.Rows
		err  error
	)

	if parentParam == "" {
		const query = `
			SELECT id, user_id, parent_id, name
			FROM folders
			WHERE user_id = $1 AND parent_id IS NULL
			ORDER BY name ASC
		`
		rows, err = h.db.QueryContext(c.Request.Context(), query, userID)
	} else {
		parentID, parseErr := strconv.ParseInt(parentParam, 10, 64)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parent_id"})
			return
		}
		const query = `
			SELECT id, user_id, parent_id, name
			FROM folders
			WHERE user_id = $1 AND parent_id = $2
			ORDER BY name ASC
		`
		rows, err = h.db.QueryContext(c.Request.Context(), query, userID, parentID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list folders"})
		return
	}
	defer rows.Close()

	resp := make([]folderResponse, 0)
	for rows.Next() {
		var item folderResponse
		if scanErr := rows.Scan(&item.ID, &item.UserID, &item.ParentID, &item.Name); scanErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan folders"})
			return
		}
		resp = append(resp, item)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list folders"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": resp})
}

// Rename handles PATCH /api/folders/:id.
func (h *Handler) Rename(c *gin.Context) {
	userID, ok := authUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	folderID, ok := folderIDParam(c)
	if !ok {
		return
	}

	var req renameFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "folder name is required"})
		return
	}

	const query = `
		UPDATE folders
		SET name = $1
		WHERE id = $2 AND user_id = $3
		RETURNING id, user_id, parent_id, name
	`
	var out folderResponse
	err := h.db.QueryRowContext(c.Request.Context(), query, name, folderID, userID).
		Scan(&out.ID, &out.UserID, &out.ParentID, &out.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "folder not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to rename folder"})
		return
	}

	c.JSON(http.StatusOK, out)
}

// Delete handles DELETE /api/folders/:id.
func (h *Handler) Delete(c *gin.Context) {
	userID, ok := authUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	folderID, ok := folderIDParam(c)
	if !ok {
		return
	}

	const query = `
		DELETE FROM folders
		WHERE id = $1 AND user_id = $2
	`
	result, err := h.db.ExecContext(c.Request.Context(), query, folderID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete folder"})
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "folder not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) folderBelongsToUser(c *gin.Context, userID string, folderID int64) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM folders
			WHERE id = $1 AND user_id = $2
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

func folderIDParam(c *gin.Context) (int64, bool) {
	rawID := strings.TrimSpace(c.Param("id"))
	folderID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || folderID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid folder id"})
		return 0, false
	}
	return folderID, true
}
