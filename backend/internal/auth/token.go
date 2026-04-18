package auth

import (
	"crypto/rand"
	"encoding/base64"
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
