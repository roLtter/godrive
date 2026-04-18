package auth

import (
	"testing"
	"time"
)

func TestTokenIssuer_IssueAndValidateAccessToken(t *testing.T) {
	issuer := NewTokenIssuer("test-secret", 15, 60)

	token, expiresAt, err := issuer.IssueAccessToken("user-123", "user@example.com")
	if err != nil {
		t.Fatalf("IssueAccessToken returned error: %v", err)
	}
	if token == "" {
		t.Fatalf("expected non-empty token")
	}
	if expiresAt.Before(time.Now().UTC()) {
		t.Fatalf("expected future expiration, got %v", expiresAt)
	}

	claims, err := issuer.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken returned error: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Fatalf("expected user id user-123, got %s", claims.UserID)
	}
	if claims.Email != "user@example.com" {
		t.Fatalf("expected email user@example.com, got %s", claims.Email)
	}
	if claims.Exp.Before(time.Now().UTC()) {
		t.Fatalf("expected claims exp in future, got %v", claims.Exp)
	}
}

func TestTokenIssuer_ValidateAccessToken_WrongSecret(t *testing.T) {
	issuerA := NewTokenIssuer("secret-a", 15, 60)
	issuerB := NewTokenIssuer("secret-b", 15, 60)

	token, _, err := issuerA.IssueAccessToken("user-123", "user@example.com")
	if err != nil {
		t.Fatalf("IssueAccessToken returned error: %v", err)
	}

	if _, err := issuerB.ValidateAccessToken(token); err == nil {
		t.Fatalf("expected validation error with wrong secret")
	}
}

func TestTokenIssuer_ValidateAccessToken_Expired(t *testing.T) {
	issuer := NewTokenIssuer("test-secret", -1, 60)

	token, _, err := issuer.IssueAccessToken("user-123", "user@example.com")
	if err != nil {
		t.Fatalf("IssueAccessToken returned error: %v", err)
	}

	if _, err := issuer.ValidateAccessToken(token); err == nil {
		t.Fatalf("expected validation error for expired token")
	}
}
