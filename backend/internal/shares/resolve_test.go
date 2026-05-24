package shares

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"
	"time"

	"cloudstore/backend/internal/db/postgres"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func TestResolve_SuccessNoPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	storage := &resolveStorage{
		url: mustURL(t, "https://minio.example/presigned"),
	}
	h := NewHandler(&postgres.Client{DB: sqlDB}, storage)

	r := gin.New()
	r.GET("/s/:token", h.Resolve)

	token := "550e8400-e29b-41d4-a716-446655440000"
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT f.s3_key, s.expires_at, s.password_hash
		FROM shares s
		INNER JOIN files f ON f.id = s.file_id
		WHERE s.token = $1
		  AND f.deleted_at IS NULL
		LIMIT 1
	`)).
		WithArgs(token).
		WillReturnRows(sqlmock.NewRows([]string{"s3_key", "expires_at", "password_hash"}).
			AddRow("users/1/a.txt", time.Now().UTC().Add(5*time.Minute), nil))
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE shares
		SET download_count = download_count + 1,
		    last_accessed_at = NOW()
		WHERE token = $1
	`)).
		WithArgs(token).
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodGet, "/s/"+token, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status: want %d got %d body=%s", http.StatusFound, rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Location") != "https://minio.example/presigned" {
		t.Fatalf("unexpected location: %s", rec.Header().Get("Location"))
	}
	if storage.lastKey != "users/1/a.txt" {
		t.Fatalf("storage key: want users/1/a.txt got %s", storage.lastKey)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestResolve_RequiresPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}

	h := NewHandler(&postgres.Client{DB: sqlDB}, &resolveStorage{url: mustURL(t, "https://minio.example/presigned")})
	r := gin.New()
	r.GET("/s/:token", h.Resolve)

	token := "550e8400-e29b-41d4-a716-446655440000"
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT f.s3_key, s.expires_at, s.password_hash
		FROM shares s
		INNER JOIN files f ON f.id = s.file_id
		WHERE s.token = $1
		  AND f.deleted_at IS NULL
		LIMIT 1
	`)).
		WithArgs(token).
		WillReturnRows(sqlmock.NewRows([]string{"s3_key", "expires_at", "password_hash"}).
			AddRow("users/1/a.txt", time.Now().UTC().Add(5*time.Minute), string(hash)))

	req := httptest.NewRequest(http.MethodGet, "/s/"+token, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: want %d got %d body=%s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestResolve_Expired(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	h := NewHandler(&postgres.Client{DB: sqlDB}, &resolveStorage{url: mustURL(t, "https://minio.example/presigned")})
	r := gin.New()
	r.GET("/s/:token", h.Resolve)

	token := "550e8400-e29b-41d4-a716-446655440000"
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT f.s3_key, s.expires_at, s.password_hash
		FROM shares s
		INNER JOIN files f ON f.id = s.file_id
		WHERE s.token = $1
		  AND f.deleted_at IS NULL
		LIMIT 1
	`)).
		WithArgs(token).
		WillReturnRows(sqlmock.NewRows([]string{"s3_key", "expires_at", "password_hash"}).
			AddRow("users/1/a.txt", time.Now().UTC().Add(-1*time.Minute), nil))

	req := httptest.NewRequest(http.MethodGet, "/s/"+token, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusGone {
		t.Fatalf("status: want %d got %d body=%s", http.StatusGone, rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestResolve_SuccessWithPassword_TracksAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}

	storage := &resolveStorage{url: mustURL(t, "https://minio.example/presigned")}
	h := NewHandler(&postgres.Client{DB: sqlDB}, storage)
	r := gin.New()
	r.GET("/s/:token", h.Resolve)

	token := "550e8400-e29b-41d4-a716-446655440000"
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT f.s3_key, s.expires_at, s.password_hash
		FROM shares s
		INNER JOIN files f ON f.id = s.file_id
		WHERE s.token = $1
		  AND f.deleted_at IS NULL
		LIMIT 1
	`)).
		WithArgs(token).
		WillReturnRows(sqlmock.NewRows([]string{"s3_key", "expires_at", "password_hash"}).
			AddRow("users/1/a.txt", time.Now().UTC().Add(5*time.Minute), string(hash)))
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE shares
		SET download_count = download_count + 1,
		    last_accessed_at = NOW()
		WHERE token = $1
	`)).
		WithArgs(token).
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodGet, "/s/"+token+"?password=secret123", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status: want %d got %d body=%s", http.StatusFound, rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

type resolveStorage struct {
	url     *url.URL
	lastKey string
	err     error
}

func (s *resolveStorage) PresignedGetURL(_ context.Context, objectName string) (*url.URL, error) {
	s.lastKey = objectName
	if s.err != nil {
		return nil, s.err
	}
	return s.url, nil
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse: %v", err)
	}
	return u
}
