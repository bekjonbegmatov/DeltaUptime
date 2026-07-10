package auth

import (
	"testing"
	"time"
)

func TestTokenManagerIssueAndParseAccessToken(t *testing.T) {
	manager, err := NewTokenManager("access-secret", "refresh-secret", 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("NewTokenManager: %v", err)
	}

	token, _, err := manager.IssueAccessToken("user-123")
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}

	userID, err := manager.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("ParseAccessToken: %v", err)
	}
	if userID != "user-123" {
		t.Fatalf("ParseAccessToken userID = %q, want user-123", userID)
	}
}

func TestTokenManagerGenerateRefreshToken(t *testing.T) {
	manager, err := NewTokenManager("access-secret", "refresh-secret", 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("NewTokenManager: %v", err)
	}

	token, hash, expiresAt, err := manager.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}
	if token == "" || hash == "" {
		t.Fatal("expected token and hash to be non-empty")
	}
	if manager.HashRefreshToken(token) != hash {
		t.Fatal("HashRefreshToken output does not match stored hash")
	}
	if !expiresAt.After(time.Now()) {
		t.Fatal("expected refresh token expiry in the future")
	}
}
