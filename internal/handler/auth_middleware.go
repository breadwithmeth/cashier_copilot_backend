package handler

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	"cashier_copilot_backend/internal/model"
	"cashier_copilot_backend/internal/service"
)

type authContextKey string

const currentUserKey authContextKey = "current_user"

// RequireAuth validates a Bearer token and optionally restricts allowed roles.
func RequireAuth(auth *service.AuthService, roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, role := range roles {
		allowed[role] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r.Header.Get("Authorization"))
			if token == "" {
				writeJSON(w, http.StatusUnauthorized, model.ErrorResponse{Error: "missing bearer token"})
				return
			}

			user, err := auth.ValidateAccessToken(r.Context(), token)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, model.ErrorResponse{Error: "invalid or expired token"})
				return
			}

			if len(allowed) > 0 && !allowed[user.Role] {
				writeJSON(w, http.StatusForbidden, model.ErrorResponse{Error: "forbidden"})
				return
			}

			ctx := context.WithValue(r.Context(), currentUserKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAPIKey validates service-to-service API key for POS webhooks.
func RequireAPIKey(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			provided := r.Header.Get("X-API-Key")
			if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(apiKey)) != 1 {
				writeJSON(w, http.StatusUnauthorized, model.ErrorResponse{Error: "invalid api key"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CurrentUser returns the authenticated user from request context.
func CurrentUser(r *http.Request) *model.AuthUser {
	user, _ := r.Context().Value(currentUserKey).(*model.AuthUser)
	return user
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}
