package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"cashier_copilot_backend/internal/model"
	"cashier_copilot_backend/internal/service"
)

// AuthHandler handles login and current-user endpoints.
type AuthHandler struct {
	auth   *service.AuthService
	logger *slog.Logger
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(auth *service.AuthService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{auth: auth, logger: logger}
}

// HandleLogin authenticates a user and returns an access token.
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Error:   "invalid request body",
			Details: err.Error(),
		})
		return
	}

	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Error: "username and password are required",
		})
		return
	}

	user, token, expiresAt, err := h.auth.Authenticate(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			writeJSON(w, http.StatusUnauthorized, model.ErrorResponse{Error: "invalid credentials"})
			return
		}
		h.logger.Error("auth_handler: login failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{Error: "login failed"})
		return
	}

	writeJSON(w, http.StatusOK, model.LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
		User:        *user,
	})
}

// HandleMe returns the current authenticated user.
func (h *AuthHandler) HandleMe(w http.ResponseWriter, r *http.Request) {
	user := CurrentUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, model.ErrorResponse{Error: "unauthorized"})
		return
	}
	writeJSON(w, http.StatusOK, user)
}
