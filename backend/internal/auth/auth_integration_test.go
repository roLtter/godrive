package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	redisClient "cloudstore/backend/internal/cache/redis"
	"cloudstore/backend/internal/db/postgres"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type integrationLoginResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	AccessExpiresAt  string `json:"access_expires_at"`
	RefreshExpiresAt string `json:"refresh_expires_at"`
}

type integrationRefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func TestIntegration_RegisterLoginRefreshFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	miniRedis, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer miniRedis.Close()

	redisDB, err := redisClient.New(context.Background(), redisClient.Config{
		URL:          "redis://" + miniRedis.Addr() + "/0",
		DialTimeout:  time.Second,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
		PoolSize:     5,
		MinIdleConns: 1,
	})
	if err != nil {
		t.Fatalf("failed to create redis client: %v", err)
	}
	defer func() { _ = redisDB.Close() }()

	pgClient := &postgres.Client{DB: sqlDB}
	refreshStore := NewRefreshStore(redisDB)
	tokenIssuer := NewTokenIssuer("integration-secret", 15, 60)

	registerHandler := NewRegisterHandler(pgClient)
	loginHandler := NewLoginHandler(pgClient, tokenIssuer, refreshStore)
	refreshHandler := NewRefreshHandler(pgClient, tokenIssuer, refreshStore)

	router := gin.New()
	router.POST("/register", registerHandler.Handle)
	router.POST("/login", loginHandler.Handle)
	router.POST("/refresh", refreshHandler.Handle)

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("Password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs("user@example.com", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "created_at"}).
			AddRow("user-1", "user@example.com", time.Now().UTC()))

	mock.ExpectQuery(`SELECT id, email, password_hash`).
		WithArgs("user@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash"}).
			AddRow("user-1", "user@example.com", string(hashedPassword)))

	mock.ExpectQuery(`SELECT email`).
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"email"}).
			AddRow("user@example.com"))

	registerBody := `{"email":"user@example.com","password":"Password123"}`
	registerReq := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()
	router.ServeHTTP(registerRec, registerReq)
	if registerRec.Code != http.StatusCreated {
		t.Fatalf("expected register status 201, got %d, body=%s", registerRec.Code, registerRec.Body.String())
	}

	loginBody := `{"email":"user@example.com","password":"Password123"}`
	loginReq := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d, body=%s", loginRec.Code, loginRec.Body.String())
	}

	var loginResp integrationLoginResponse
	if err := json.Unmarshal(loginRec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("failed to parse login response: %v", err)
	}
	if loginResp.AccessToken == "" || loginResp.RefreshToken == "" {
		t.Fatalf("expected non-empty login tokens")
	}

	refreshReqBody, _ := json.Marshal(map[string]string{"refresh_token": loginResp.RefreshToken})
	refreshReq := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBuffer(refreshReqBody))
	refreshReq.Header.Set("Content-Type", "application/json")
	refreshRec := httptest.NewRecorder()
	router.ServeHTTP(refreshRec, refreshReq)
	if refreshRec.Code != http.StatusOK {
		t.Fatalf("expected refresh status 200, got %d, body=%s", refreshRec.Code, refreshRec.Body.String())
	}

	var refreshResp integrationRefreshResponse
	if err := json.Unmarshal(refreshRec.Body.Bytes(), &refreshResp); err != nil {
		t.Fatalf("failed to parse refresh response: %v", err)
	}
	if refreshResp.AccessToken == "" || refreshResp.RefreshToken == "" {
		t.Fatalf("expected non-empty refresh response tokens")
	}
	if refreshResp.RefreshToken == loginResp.RefreshToken {
		t.Fatalf("expected refresh token rotation")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

