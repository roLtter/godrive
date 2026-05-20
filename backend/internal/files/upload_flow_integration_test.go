package files

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"cloudstore/backend/internal/auth"
	"cloudstore/backend/internal/db/postgres"
	"cloudstore/backend/internal/middleware"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

// minimal JPEG header so http.DetectContentType returns image/jpeg
var testJPEGBytes = []byte{
	0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0x01, 0x01, 0x01, 0x00, 0x48, 0x00, 0x48, 0x00, 0x00,
	0xff, 0xdb, 0x00, 0x43, 0x00,
}

type fakeObjectStorage struct {
	putKeys    []string
	removeKeys []string
}

func (f *fakeObjectStorage) PutObject(_ context.Context, objectName string, r io.Reader, _ int64, _ string) error {
	f.putKeys = append(f.putKeys, objectName)
	_, err := io.Copy(io.Discard, r)
	return err
}

func (f *fakeObjectStorage) PresignedGetURL(_ context.Context, objectName string) (*url.URL, error) {
	return url.Parse("https://example.test/presigned?object=" + url.PathEscape(objectName))
}

func (f *fakeObjectStorage) RemoveObject(_ context.Context, objectName string) error {
	f.removeKeys = append(f.removeKeys, objectName)
	return nil
}

func TestIntegration_UploadDownloadDeleteFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { uploadObjectKeyHook = nil })

	const (
		jwtSecret = "integration-files-secret"
		userID    = "11111111-1111-1111-1111-111111111111"
		folderID  = int64(1)
		fileID    = int64(101)
	)

	fixedS3Key := userID + "/integration-test-object.bin"
	uploadObjectKeyHook = func(_, _ string) (string, error) { return fixedS3Key, nil }

	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	fileSize := int64(len(testJPEGBytes))

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
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
		WithArgs(userID, folderID, "test.jpg", fileSize, "image/jpeg", fixedS3Key).
		WillReturnRows(sqlmock.NewRows([]string{"id", "folder_id", "name", "size", "mime", "s3_key", "created_at"}).
			AddRow(fileID, folderID, "test.jpg", fileSize, "image/jpeg", fixedS3Key, createdAt))
	mock.ExpectExec(`UPDATE users`).
		WithArgs(fileSize, userID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	mock.ExpectQuery(`SELECT s3_key`).
		WithArgs(fileID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"s3_key"}).AddRow(fixedS3Key))

	mock.ExpectExec(`UPDATE files`).
		WithArgs(fileID, userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	db := &postgres.Client{DB: sqlDB}
	stub := &fakeObjectStorage{}
	h := NewHandler(db, stub, 10<<20, []string{"image/jpeg"})

	issuer := auth.NewTokenIssuer(jwtSecret, 15, 60)
	token, _, err := issuer.IssueAccessToken(userID, "u@example.com")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	r := gin.New()
	api := r.Group("/api")
	api.Use(middleware.JWTAuth(jwtSecret))
	api.POST("/upload", h.Upload)
	api.GET("/download", h.Download)
	api.DELETE("/files/:id", h.SoftDelete)

	// --- upload ---
	body := &bytes.Buffer{}
	mp := multipart.NewWriter(body)
	_ = mp.WriteField("folder_id", strconv.FormatInt(folderID, 10))
	part, err := mp.CreateFormFile("file", "test.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(testJPEGBytes); err != nil {
		t.Fatal(err)
	}
	if err := mp.Close(); err != nil {
		t.Fatal(err)
	}

	uploadReq := httptest.NewRequest(http.MethodPost, "/api/upload", body)
	uploadReq.Header.Set("Content-Type", mp.FormDataContentType())
	uploadReq.Header.Set("Authorization", "Bearer "+token)
	uploadRec := httptest.NewRecorder()
	r.ServeHTTP(uploadRec, uploadReq)
	if uploadRec.Code != http.StatusCreated {
		t.Fatalf("upload: want 201, got %d body=%s", uploadRec.Code, uploadRec.Body.String())
	}
	var up uploadResponse
	if err := json.Unmarshal(uploadRec.Body.Bytes(), &up); err != nil {
		t.Fatalf("upload json: %v", err)
	}
	if up.ID != fileID {
		t.Fatalf("upload id: want %d, got %d", fileID, up.ID)
	}
	if up.S3Key != fixedS3Key {
		t.Fatalf("upload s3_key: want %q, got %q", fixedS3Key, up.S3Key)
	}
	if len(stub.putKeys) != 1 || stub.putKeys[0] != fixedS3Key {
		t.Fatalf("PutObject key: want %q once, got %#v", fixedS3Key, stub.putKeys)
	}

	// --- download ---
	dlReq := httptest.NewRequest(http.MethodGet, "/api/download?file_id="+strconv.FormatInt(fileID, 10), nil)
	dlReq.Header.Set("Authorization", "Bearer "+token)
	dlRec := httptest.NewRecorder()
	r.ServeHTTP(dlRec, dlReq)
	if dlRec.Code != http.StatusFound {
		t.Fatalf("download: want 302, got %d body=%s", dlRec.Code, dlRec.Body.String())
	}
	loc := dlRec.Header().Get("Location")
	if !strings.HasPrefix(loc, "https://example.test/presigned") {
		t.Fatalf("download Location: %q", loc)
	}
	if !strings.Contains(loc, url.QueryEscape(fixedS3Key)) {
		t.Fatalf("download Location should contain object: %q", loc)
	}

	// --- soft delete ---
	delReq := httptest.NewRequest(http.MethodDelete, "/api/files/"+strconv.FormatInt(fileID, 10), nil)
	delReq.Header.Set("Authorization", "Bearer "+token)
	delRec := httptest.NewRecorder()
	r.ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusNoContent {
		t.Fatalf("delete: want 204, got %d body=%s", delRec.Code, delRec.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
