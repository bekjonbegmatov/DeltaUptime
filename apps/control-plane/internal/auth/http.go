package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

type HandlerService interface {
	Login(ctx context.Context, input LoginInput) (AuthResult, error)
	Me(ctx context.Context, accessToken string) (AuthResult, error)
	Refresh(ctx context.Context, input RefreshInput) (AuthResult, error)
	Register(ctx context.Context, input RegisterInput) (AuthResult, error)
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
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
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

func (h *Handler) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrEmailTaken), errors.Is(err, ErrOrganizationSlugTaken):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrRefreshTokenInvalid), errors.Is(err, ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, err.Error())
	default:
		h.log.Error("auth request failed", "error", err)
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
