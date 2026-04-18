package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenIssuer creates signed JWT access tokens.
type TokenIssuer struct {
	secretKey []byte
	accessTTL time.Duration
	refreshTTL time.Duration
}

// AccessClaims represents validated access token claims.
type AccessClaims struct {
	UserID string
	Email  string
	Exp    time.Time
	Iat    time.Time
}

// NewTokenIssuer builds a new JWT issuer.
func NewTokenIssuer(secret string, accessTTLMinutes, refreshTTLMinutes int) *TokenIssuer {
	return &TokenIssuer{
		secretKey: []byte(secret),
		accessTTL: time.Duration(accessTTLMinutes) * time.Minute,
		refreshTTL: time.Duration(refreshTTLMinutes) * time.Minute,
	}
}

// IssueAccessToken generates a signed access token for the user.
func (t *TokenIssuer) IssueAccessToken(userID, email string) (string, time.Time, error) {
	expiresAt := time.Now().UTC().Add(t.accessTTL)
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"exp":   expiresAt.Unix(),
		"iat":   time.Now().UTC().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(t.secretKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("issue access token: %w", err)
	}

	return signed, expiresAt, nil
}

// ValidateAccessToken parses and validates access token signature and claims.
func (t *TokenIssuer) ValidateAccessToken(tokenString string) (AccessClaims, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		return t.secretKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		return AccessClaims{}, fmt.Errorf("validate access token: %w", err)
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return AccessClaims{}, errors.New("validate access token: missing sub claim")
	}
	email, _ := claims["email"].(string)

	expValue, ok := claims["exp"].(float64)
	if !ok {
		return AccessClaims{}, errors.New("validate access token: missing exp claim")
	}
	iatValue, ok := claims["iat"].(float64)
	if !ok {
		return AccessClaims{}, errors.New("validate access token: missing iat claim")
	}

	return AccessClaims{
		UserID: sub,
		Email:  email,
		Exp:    time.Unix(int64(expValue), 0).UTC(),
		Iat:    time.Unix(int64(iatValue), 0).UTC(),
	}, nil
}

// IssueRefreshToken generates a random opaque refresh token.
func (t *TokenIssuer) IssueRefreshToken() (string, time.Time, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", time.Time{}, fmt.Errorf("issue refresh token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	expiresAt := time.Now().UTC().Add(t.refreshTTL)
	return token, expiresAt, nil
}
