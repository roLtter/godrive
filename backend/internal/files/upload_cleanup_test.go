package files

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"cloudstore/backend/internal/auth"
	"cloudstore/backend/internal/db/postgres"
	"cloudstore/backend/internal/middleware"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func TestUpload_RemovesMinIOObjectWhenDBReserveFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { uploadObjectKeyHook = nil })

	const (
		jwtSecret = "cleanup-test-secret"
		userID    = "22222222-2222-2222-2222-222222222222"
		folderID  = int64(1)
	)
	fixedKey := userID + "/fixed-cleanup-key.bin"
	uploadObjectKeyHook = func(_, _ string) (string, error) { return fixedKey, nil }

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(folderID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT storage_used_bytes`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"storage_used_bytes", "storage_quota_bytes"}).AddRow(int64(0), int64(1<<30)))
	mock.ExpectQuery(`INSERT INTO files`).
		WithArgs(userID, folderID, "x.jpg", int64(len(testJPEGBytes)), "image/jpeg", fixedKey).
		WillReturnError(fmt.Errorf("simulated db failure"))
	mock.ExpectRollback()

	stub := &fakeObjectStorage{}
	h := NewHandler(&postgres.Client{DB: sqlDB}, stub, 10<<20, []string{"image/jpeg"})

	issuer := auth.NewTokenIssuer(jwtSecret, 15, 60)
	token, _, err := issuer.IssueAccessToken(userID, "u@example.com")
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	api := r.Group("/api")
	api.Use(middleware.JWTAuth(jwtSecret))
	api.POST("/upload", h.Upload)

	body := &bytes.Buffer{}
	mp := multipart.NewWriter(body)
	_ = mp.WriteField("folder_id", strconv.FormatInt(folderID, 10))
	part, _ := mp.CreateFormFile("file", "x.jpg")
	_, _ = part.Write(testJPEGBytes)
	_ = mp.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/upload", body)
	req.Header.Set("Content-Type", mp.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d body=%s", rec.Code, rec.Body.String())
	}
	if len(stub.putKeys) != 1 || stub.putKeys[0] != fixedKey {
		t.Fatalf("PutObject: got %#v", stub.putKeys)
	}
	if len(stub.removeKeys) != 1 || stub.removeKeys[0] != fixedKey {
		t.Fatalf("RemoveObject cleanup: got %#v", stub.removeKeys)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
