package shares

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"cloudstore/backend/internal/db/postgres"
	"cloudstore/backend/internal/middleware"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func TestListActive_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	h := NewHandler(&postgres.Client{DB: sqlDB}, &resolveStorage{url: mustURL(t, "https://example.test")})
	r := gin.New()
	secret := "shares-list-secret"
	api := r.Group("/api")
	api.Use(middleware.JWTAuth(secret))
	api.GET("/shares", h.ListActive)

	userID := "11111111-1111-1111-1111-111111111111"
	expires := time.Now().UTC().Add(time.Hour)
	lastSeen := time.Now().UTC().Add(-time.Minute)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT s.id, s.file_id, s.token, s.expires_at, s.download_count, s.last_accessed_at
		FROM shares s
		INNER JOIN files f ON f.id = s.file_id
		WHERE f.user_id = $1
		  AND f.deleted_at IS NULL
		  AND s.expires_at > NOW()
		ORDER BY s.expires_at ASC
	`)).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "file_id", "token", "expires_at", "download_count", "last_accessed_at"}).
			AddRow(int64(10), int64(42), "550e8400-e29b-41d4-a716-446655440000", expires, int64(3), lastSeen))

	req := httptest.NewRequest(http.MethodGet, "/api/shares", nil)
	req.Header.Set("Authorization", "Bearer "+testToken(t, secret, userID))
	req.Host = "localhost:8080"
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want %d got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var payload struct {
		Items []shareListItem `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("items len: want 1 got %d", len(payload.Items))
	}
	if payload.Items[0].URL != "http://localhost:8080/s/550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("unexpected share url: %s", payload.Items[0].URL)
	}
	if payload.Items[0].DownloadCount != 3 {
		t.Fatalf("download_count: want 3 got %d", payload.Items[0].DownloadCount)
	}
	if payload.Items[0].LastAccessedAt == nil {
		t.Fatal("last_accessed_at should be present")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestRevoke_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	h := NewHandler(&postgres.Client{DB: sqlDB}, &resolveStorage{url: mustURL(t, "https://example.test")})
	r := gin.New()
	secret := "shares-revoke-secret"
	api := r.Group("/api")
	api.Use(middleware.JWTAuth(secret))
	api.DELETE("/shares/:id", h.Revoke)

	userID := "11111111-1111-1111-1111-111111111111"
	mock.ExpectExec(regexp.QuoteMeta(`
		DELETE FROM shares s
		USING files f
		WHERE s.id = $1
		  AND f.id = s.file_id
		  AND f.user_id = $2
	`)).
		WithArgs(int64(10), userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodDelete, "/api/shares/10", nil)
	req.Header.Set("Authorization", "Bearer "+testToken(t, secret, userID))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: want %d got %d body=%s", http.StatusNoContent, rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestRevoke_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	h := NewHandler(&postgres.Client{DB: sqlDB}, &resolveStorage{url: mustURL(t, "https://example.test")})
	r := gin.New()
	secret := "shares-revoke-secret"
	api := r.Group("/api")
	api.Use(middleware.JWTAuth(secret))
	api.DELETE("/shares/:id", h.Revoke)

	userID := "11111111-1111-1111-1111-111111111111"
	mock.ExpectExec(regexp.QuoteMeta(`
		DELETE FROM shares s
		USING files f
		WHERE s.id = $1
		  AND f.id = s.file_id
		  AND f.user_id = $2
	`)).
		WithArgs(int64(999), userID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	req := httptest.NewRequest(http.MethodDelete, "/api/shares/999", nil)
	req.Header.Set("Authorization", "Bearer "+testToken(t, secret, userID))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: want %d got %d body=%s", http.StatusNotFound, rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

