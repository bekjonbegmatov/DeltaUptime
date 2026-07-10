package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenManager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

type accessClaims struct {
	jwt.RegisteredClaims
}

func NewTokenManager(accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration) (*TokenManager, error) {
	if accessSecret == "" || refreshSecret == "" {
		return nil, fmt.Errorf("%w: access and refresh secrets are required", ErrInvalidInput)
	}
	if accessTTL <= 0 || refreshTTL <= 0 {
		return nil, fmt.Errorf("%w: token TTLs must be positive", ErrInvalidInput)
	}

	return &TokenManager{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
	}, nil
}

func (m *TokenManager) IssueAccessToken(userID string) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(m.accessTTL)

	claims := accessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.accessSecret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}

	return signed, expiresAt, nil
}

func (m *TokenManager) ParseAccessToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &accessClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("%w: unexpected signing method", ErrUnauthorized)
		}
		return m.accessSecret, nil
	})
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(*accessClaims)
	if !ok || !token.Valid || claims.Subject == "" {
		return "", ErrUnauthorized
	}

	return claims.Subject, nil
}

func (m *TokenManager) GenerateRefreshToken() (string, string, time.Time, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", time.Time{}, fmt.Errorf("generate refresh token: %w", err)
	}

	token := base64.RawURLEncoding.EncodeToString(raw)
	expiresAt := time.Now().UTC().Add(m.refreshTTL)
	return token, m.HashRefreshToken(token), expiresAt, nil
}

func (m *TokenManager) HashRefreshToken(token string) string {
	mac := hmac.New(sha256.New, m.refreshSecret)
	_, _ = mac.Write([]byte(token))
	return hex.EncodeToString(mac.Sum(nil))
}
