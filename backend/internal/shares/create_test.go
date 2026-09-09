package shares

import (
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/json"
	"net/url"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"cloudstore/backend/internal/db/postgres"
	"cloudstore/backend/internal/middleware"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestCreateShare_Success(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	client := &postgres.Client{DB: db}
	h := NewHandler(client, &fakeStorage{})

	router := gin.New()
	secret := "test-secret"
	api := router.Group("/api")
	api.Use(middleware.JWTAuth(secret))
	api.POST("/shares", h.Create)

	userID := "11111111-1111-1111-1111-111111111111"
	fileID := int64(42)
	exp := time.Now().UTC().Add(30 * 24 * time.Hour).Round(0)

	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO shares (file_id, token, expires_at, password_hash)
		SELECT f.id, $2, $3, $4
		FROM files f
		WHERE f.id = $1
		  AND f.user_id = $5
		  AND f.deleted_at IS NULL
		RETURNING id, file_id, token, expires_at
	`)).
		WithArgs(fileID, uuidLikeArg{}, anyTimeArg{}, nil, userID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "file_id", "token", "expires_at"}).
			AddRow(int64(7), fileID, "550e8400-e29b-41d4-a716-446655440000", exp))

	body := bytes.NewBufferString(`{"file_id":42}`)
	req := httptest.NewRequest(http.MethodPost, "/api/shares", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testToken(t, secret, userID))
	req.Host = "localhost:8080"

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status: want %d, got %d body=%s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var got createShareResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if got.FileID != fileID {
		t.Fatalf("file_id: want %d, got %d", fileID, got.FileID)
	}
	if !regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`).MatchString(got.Token) {
		t.Fatalf("token is not UUIDv4: %q", got.Token)
	}
	if got.URL != "http://localhost:8080/s/"+got.Token {
		t.Fatalf("url: got %q", got.URL)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestCreateShare_WithTTLAndPassword(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	client := &postgres.Client{DB: db}
	h := NewHandler(client, &fakeStorage{})

	router := gin.New()
	secret := "test-secret"
	api := router.Group("/api")
	api.Use(middleware.JWTAuth(secret))
	api.POST("/shares", h.Create)

	userID := "11111111-1111-1111-1111-111111111111"
	fileID := int64(42)
	exp := time.Now().UTC().Add(3600 * time.Second).Round(0)

	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO shares (file_id, token, expires_at, password_hash)
		SELECT f.id, $2, $3, $4
		FROM files f
		WHERE f.id = $1
		  AND f.user_id = $5
		  AND f.deleted_at IS NULL
		RETURNING id, file_id, token, expires_at
	`)).
		WithArgs(fileID, uuidLikeArg{}, anyTimeArg{}, passwordHashArg{}, userID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "file_id", "token", "expires_at"}).
			AddRow(int64(8), fileID, "550e8400-e29b-41d4-a716-446655440000", exp))

	body := bytes.NewBufferString(`{"file_id":42,"ttl_seconds":3600,"password":"secret123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/shares", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testToken(t, secret, userID))
	req.Host = "localhost:8080"

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status: want %d, got %d body=%s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestCreateShare_WithExpiresInAlias(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	client := &postgres.Client{DB: db}
	h := NewHandler(client, &fakeStorage{})

	router := gin.New()
	secret := "test-secret"
	api := router.Group("/api")
	api.Use(middleware.JWTAuth(secret))
	api.POST("/shares", h.Create)

	userID := "11111111-1111-1111-1111-111111111111"
	fileID := int64(42)
	exp := time.Now().UTC().Add(86400 * time.Second).Round(0)

	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO shares (file_id, token, expires_at, password_hash)
		SELECT f.id, $2, $3, $4
		FROM files f
		WHERE f.id = $1
		  AND f.user_id = $5
		  AND f.deleted_at IS NULL
		RETURNING id, file_id, token, expires_at
	`)).
		WithArgs(fileID, uuidLikeArg{}, anyTimeArg{}, nil, userID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "file_id", "token", "expires_at"}).
			AddRow(int64(9), fileID, "550e8400-e29b-41d4-a716-446655440000", exp))

	body := bytes.NewBufferString(`{"file_id":42,"expires_in":86400}`)
	req := httptest.NewRequest(http.MethodPost, "/api/shares", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testToken(t, secret, userID))
	req.Host = "localhost:8080"

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status: want %d, got %d body=%s", http.StatusCreated, rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestCreateShare_InvalidTTL(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	client := &postgres.Client{DB: db}
	h := NewHandler(client, &fakeStorage{})

	router := gin.New()
	secret := "test-secret"
	api := router.Group("/api")
	api.Use(middleware.JWTAuth(secret))
	api.POST("/shares", h.Create)

	userID := "11111111-1111-1111-1111-111111111111"
	body := bytes.NewBufferString(`{"file_id":42,"ttl_seconds":0}`)
	req := httptest.NewRequest(http.MethodPost, "/api/shares", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testToken(t, secret, userID))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: want %d, got %d body=%s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func TestCreateShare_ShortPassword(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	client := &postgres.Client{DB: db}
	h := NewHandler(client, &fakeStorage{})

	router := gin.New()
	secret := "test-secret"
	api := router.Group("/api")
	api.Use(middleware.JWTAuth(secret))
	api.POST("/shares", h.Create)

	userID := "11111111-1111-1111-1111-111111111111"
	body := bytes.NewBufferString(`{"file_id":42,"password":"123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/shares", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testToken(t, secret, userID))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: want %d, got %d body=%s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

type anyTimeArg struct{}

func (a anyTimeArg) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}

type uuidLikeArg struct{}

func (a uuidLikeArg) Match(v driver.Value) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	return regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`).MatchString(s)
}

type passwordHashArg struct{}

func (a passwordHashArg) Match(v driver.Value) bool {
	s, ok := v.(string)
	if !ok || s == "" {
		return false
	}
	return len(s) > 20
}

type fakeStorage struct{}

func (f *fakeStorage) PresignedGetURL(_ context.Context, objectName string) (*url.URL, error) {
	return url.Parse("https://example.test/get/" + objectName)
}

func testToken(t *testing.T, secret, userID string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
	})
	s, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}
	return s
}
