package auth

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeHandlerService struct{}

func (fakeHandlerService) Login(context.Context, LoginInput) (AuthResult, error) {
	return AuthResult{
		AccessToken: "access",
		User:        UserInfo{ID: "u1", Email: "user@example.com", DisplayName: "User"},
	}, nil
}

func (fakeHandlerService) Me(context.Context, string) (AuthResult, error) {
	return AuthResult{
		User: UserInfo{ID: "u1", Email: "user@example.com", DisplayName: "User"},
	}, nil
}

func (fakeHandlerService) Refresh(context.Context, RefreshInput) (AuthResult, error) {
	return AuthResult{
		AccessToken: "access-2",
		User:        UserInfo{ID: "u1", Email: "user@example.com", DisplayName: "User"},
	}, nil
}

func (fakeHandlerService) Register(context.Context, RegisterInput) (AuthResult, error) {
	return AuthResult{
		AccessToken:           "access",
		AccessTokenExpiresAt:  time.Unix(1700000000, 0).UTC(),
		RefreshToken:          "refresh",
		RefreshTokenExpiresAt: time.Unix(1700003600, 0).UTC(),
		User:                  UserInfo{ID: "u1", Email: "user@example.com", DisplayName: "User"},
		Organizations: []OrganizationInfo{{
			ID:   "o1",
			Slug: "acme",
			Name: "Acme",
			Role: "owner",
		}},
	}, nil
}

func TestRegisterRoute(t *testing.T) {
	handler := NewHandler(slog.Default(), fakeHandlerService{})
	body := `{"email":"user@example.com","password":"supersecret1","display_name":"User","organization_name":"Acme","organization_slug":"acme"}`

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q, want application/json", got)
	}
	if !strings.Contains(rec.Body.String(), `"refresh_token":"refresh"`) {
		t.Fatalf("response body missing refresh token: %s", rec.Body.String())
	}
}

func TestMeRouteRequiresBearerToken(t *testing.T) {
	handler := NewHandler(slog.Default(), fakeHandlerService{})
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}
