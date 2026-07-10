package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

type HandlerService interface {
	AddMembership(ctx context.Context, principal Principal, slug string, input MembershipCreateInput) (MembershipInfo, error)
	AuthenticateAccessToken(ctx context.Context, accessToken string) (Principal, error)
	AuthenticateAPIKey(ctx context.Context, rawAPIKey string) (Principal, error)
	BeginWebAuthnLogin(ctx context.Context, email string) (WebAuthnLoginBeginResult, error)
	BeginWebAuthnRegistration(ctx context.Context, principal Principal) (WebAuthnRegistrationBeginResult, error)
	CreateAPIKey(ctx context.Context, principal Principal, slug string, input APIKeyCreateInput) (APIKeyCreateResult, error)
	DisableTOTP(ctx context.Context, principal Principal, code string) (UserInfo, error)
	EnableTOTP(ctx context.Context, principal Principal, code string) (UserInfo, error)
	FinishWebAuthnLogin(ctx context.Context, input WebAuthnFinishInput) (AuthResult, error)
	FinishWebAuthnRegistration(ctx context.Context, principal Principal, input WebAuthnFinishInput) (WebAuthnCredentialInfo, error)
	GetOrganization(ctx context.Context, principal Principal, slug string) (OrganizationInfo, error)
	ListAPIKeys(ctx context.Context, principal Principal, slug string) ([]APIKeyInfo, error)
	ListAuditLogs(ctx context.Context, principal Principal, slug string, limit int32) ([]AuditLogInfo, error)
	ListMemberships(ctx context.Context, principal Principal, slug string) ([]MembershipInfo, error)
	ListOrganizations(ctx context.Context, principal Principal) ([]OrganizationInfo, error)
	Login(ctx context.Context, input LoginInput) (AuthResult, error)
	Me(ctx context.Context, accessToken string) (AuthResult, error)
	Refresh(ctx context.Context, input RefreshInput) (AuthResult, error)
	Register(ctx context.Context, input RegisterInput) (AuthResult, error)
	RevokeAPIKey(ctx context.Context, principal Principal, slug, apiKeyID string) (APIKeyInfo, error)
	SetupTOTP(ctx context.Context, principal Principal) (TOTPSetup, error)
	UpdateMembership(ctx context.Context, principal Principal, slug string, input MembershipUpdateInput) (MembershipInfo, error)
}

type Handler struct {
	log     *slog.Logger
	service HandlerService
}

func NewHandler(log *slog.Logger, service HandlerService) http.Handler {
	h := &Handler{
		log:     log,
		service: service,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/auth/register", h.handleRegister)
	mux.HandleFunc("POST /v1/auth/login", h.handleLogin)
	mux.HandleFunc("POST /v1/auth/refresh", h.handleRefresh)
	mux.HandleFunc("GET /v1/auth/me", h.handleMe)
	mux.HandleFunc("POST /v1/auth/totp/setup", h.handleTOTPSetup)
	mux.HandleFunc("POST /v1/auth/totp/enable", h.handleTOTPEnable)
	mux.HandleFunc("POST /v1/auth/totp/disable", h.handleTOTPDisable)
	mux.HandleFunc("POST /v1/auth/webauthn/register/begin", h.handleWebAuthnRegisterBegin)
	mux.HandleFunc("POST /v1/auth/webauthn/register/finish", h.handleWebAuthnRegisterFinish)
	mux.HandleFunc("POST /v1/auth/webauthn/login/begin", h.handleWebAuthnLoginBegin)
	mux.HandleFunc("POST /v1/auth/webauthn/login/finish", h.handleWebAuthnLoginFinish)
	mux.HandleFunc("GET /v1/organizations", h.handleOrganizations)
	mux.HandleFunc("GET /v1/organizations/{slug}", h.handleOrganization)
	mux.HandleFunc("GET /v1/organizations/{slug}/memberships", h.handleListMemberships)
	mux.HandleFunc("POST /v1/organizations/{slug}/memberships", h.handleCreateMembership)
	mux.HandleFunc("PATCH /v1/organizations/{slug}/memberships/{user_id}", h.handleUpdateMembership)
	mux.HandleFunc("GET /v1/organizations/{slug}/api-keys", h.handleListAPIKeys)
	mux.HandleFunc("POST /v1/organizations/{slug}/api-keys", h.handleCreateAPIKey)
	mux.HandleFunc("DELETE /v1/organizations/{slug}/api-keys/{api_key_id}", h.handleRevokeAPIKey)
	mux.HandleFunc("GET /v1/organizations/{slug}/audit-logs", h.handleListAuditLogs)
	return mux
}

type registerRequest struct {
	Email            string `json:"email"`
	Password         string `json:"password"`
	DisplayName      string `json:"display_name"`
	OrganizationName string `json:"organization_name"`
	OrganizationSlug string `json:"organization_slug"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	TOTPCode string `json:"totp_code"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type totpCodeRequest struct {
	Code string `json:"code"`
}

type webAuthnLoginBeginRequest struct {
	Email string `json:"email"`
}

type webAuthnFinishRequest struct {
	SessionID  string          `json:"session_id"`
	Credential json.RawMessage `json:"credential"`
}

type membershipCreateRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type membershipUpdateRequest struct {
	Role string `json:"role"`
}

type apiKeyCreateRequest struct {
	Name   string   `json:"name"`
	Scopes []string `json:"scopes"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.service.Register(r.Context(), RegisterInput(req))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.service.Login(r.Context(), LoginInput(req))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.service.Refresh(r.Context(), RefreshInput(req))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	accessToken, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing bearer token")
		return
	}

	result, err := h.service.Me(r.Context(), accessToken)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleTOTPSetup(w http.ResponseWriter, r *http.Request) {
	principal, err := h.authenticateUser(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	result, err := h.service.SetupTOTP(r.Context(), principal)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleTOTPEnable(w http.ResponseWriter, r *http.Request) {
	principal, err := h.authenticateUser(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	var req totpCodeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.service.EnableTOTP(r.Context(), principal, req.Code)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) handleTOTPDisable(w http.ResponseWriter, r *http.Request) {
	principal, err := h.authenticateUser(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	var req totpCodeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.service.DisableTOTP(r.Context(), principal, req.Code)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) handleWebAuthnRegisterBegin(w http.ResponseWriter, r *http.Request) {
	principal, err := h.authenticateUser(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	result, err := h.service.BeginWebAuthnRegistration(r.Context(), principal)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleWebAuthnRegisterFinish(w http.ResponseWriter, r *http.Request) {
	principal, err := h.authenticateUser(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	var req webAuthnFinishRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.service.FinishWebAuthnRegistration(r.Context(), principal, WebAuthnFinishInput(req))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) handleWebAuthnLoginBegin(w http.ResponseWriter, r *http.Request) {
	var req webAuthnLoginBeginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.service.BeginWebAuthnLogin(r.Context(), req.Email)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleWebAuthnLoginFinish(w http.ResponseWriter, r *http.Request) {
	var req webAuthnFinishRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.service.FinishWebAuthnLogin(r.Context(), WebAuthnFinishInput(req))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleOrganizations(w http.ResponseWriter, r *http.Request) {
	principal, err := h.authenticatePrincipal(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	orgs, err := h.service.ListOrganizations(r.Context(), principal)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, orgs)
}

func (h *Handler) handleOrganization(w http.ResponseWriter, r *http.Request) {
	principal, err := h.authenticatePrincipal(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	org, err := h.service.GetOrganization(r.Context(), principal, r.PathValue("slug"))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, org)
}

func (h *Handler) handleListMemberships(w http.ResponseWriter, r *http.Request) {
	principal, err := h.authenticatePrincipal(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	items, err := h.service.ListMemberships(r.Context(), principal, r.PathValue("slug"))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) handleCreateMembership(w http.ResponseWriter, r *http.Request) {
	principal, err := h.authenticatePrincipal(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	var req membershipCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	item, err := h.service.AddMembership(r.Context(), principal, r.PathValue("slug"), MembershipCreateInput(req))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, item)
}

func (h *Handler) handleUpdateMembership(w http.ResponseWriter, r *http.Request) {
	principal, err := h.authenticatePrincipal(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	var req membershipUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	item, err := h.service.UpdateMembership(r.Context(), principal, r.PathValue("slug"), MembershipUpdateInput{
		UserID: r.PathValue("user_id"),
		Role:   req.Role,
	})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	principal, err := h.authenticatePrincipal(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	items, err := h.service.ListAPIKeys(r.Context(), principal, r.PathValue("slug"))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	principal, err := h.authenticatePrincipal(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	var req apiKeyCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	item, err := h.service.CreateAPIKey(r.Context(), principal, r.PathValue("slug"), APIKeyCreateInput(req))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, item)
}

func (h *Handler) handleRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	principal, err := h.authenticatePrincipal(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	item, err := h.service.RevokeAPIKey(r.Context(), principal, r.PathValue("slug"), r.PathValue("api_key_id"))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) handleListAuditLogs(w http.ResponseWriter, r *http.Request) {
	principal, err := h.authenticatePrincipal(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	limit := int32(100)
	if raw := r.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = int32(value)
	}

	items, err := h.service.ListAuditLogs(r.Context(), principal, r.PathValue("slug"), limit)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) authenticatePrincipal(r *http.Request) (Principal, error) {
	if raw := strings.TrimSpace(r.Header.Get("X-API-Key")); raw != "" {
		return h.service.AuthenticateAPIKey(r.Context(), raw)
	}

	accessToken, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		return Principal{}, ErrUnauthorized
	}
	return h.service.AuthenticateAccessToken(r.Context(), accessToken)
}

func (h *Handler) authenticateUser(r *http.Request) (Principal, error) {
	principal, err := h.authenticatePrincipal(r)
	if err != nil {
		return Principal{}, err
	}
	if principal.Type != PrincipalTypeUser {
		return Principal{}, ErrUnauthorized
	}
	return principal, nil
}

func (h *Handler) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrEmailTaken), errors.Is(err, ErrOrganizationSlugTaken), errors.Is(err, ErrMembershipExists):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrPermissionDenied):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrRefreshTokenInvalid), errors.Is(err, ErrUnauthorized), errors.Is(err, ErrAPIKeyInvalid), errors.Is(err, ErrTOTPRequired), errors.Is(err, ErrTOTPInvalid):
		writeError(w, http.StatusUnauthorized, err.Error())
	default:
		h.log.Error("identity request failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func decodeJSON(r *http.Request, dst any) error {
	defer func() { _ = r.Body.Close() }()
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	return token, token != ""
}
