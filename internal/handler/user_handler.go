package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"cashier_copilot_backend/internal/model"
	"cashier_copilot_backend/internal/service"
)

// UserHandler handles admin user management endpoints.
type UserHandler struct {
	auth   *service.AuthService
	logger *slog.Logger
}

// NewUserHandler creates a UserHandler.
func NewUserHandler(auth *service.AuthService, logger *slog.Logger) *UserHandler {
	return &UserHandler{auth: auth, logger: logger}
}

// HandleListUsers returns all users. Admin only.
func (h *UserHandler) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.auth.ListUsers(r.Context())
	if err != nil {
		h.logger.Error("user_handler: failed to list users", "error", err)
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{Error: "failed to list users"})
		return
	}
	if users == nil {
		users = []model.User{}
	}
	writeJSON(w, http.StatusOK, users)
}

// HandleCreateUser creates an admin/operator/cashier user. Admin only.
func (h *UserHandler) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req model.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Error:   "invalid request body",
			Details: err.Error(),
		})
		return
	}

	if req.Username == "" || req.Password == "" || req.Role == "" {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Error: "username, password and role are required",
		})
		return
	}

	user, err := h.auth.CreateUser(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidRole):
			writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "invalid role"})
		case errors.Is(err, service.ErrUserExists):
			writeJSON(w, http.StatusConflict, model.ErrorResponse{Error: "user already exists"})
		default:
			h.logger.Error("user_handler: failed to create user", "error", err)
			writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{Error: "failed to create user"})
		}
		return
	}

	writeJSON(w, http.StatusCreated, user)
}
