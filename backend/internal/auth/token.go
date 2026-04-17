package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenIssuer creates signed JWT access tokens.
type TokenIssuer struct {
	secretKey []byte
	accessTTL time.Duration
}

// NewTokenIssuer builds a new JWT issuer.
func NewTokenIssuer(secret string, accessTTLMinutes int) *TokenIssuer {
	return &TokenIssuer{
		secretKey: []byte(secret),
		accessTTL: time.Duration(accessTTLMinutes) * time.Minute,
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
