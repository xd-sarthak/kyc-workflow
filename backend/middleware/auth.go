package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"kyc/backend/services"
)

type contextKey string

const (
	ContextUserID contextKey = "userID"
	ContextRole   contextKey = "role"
)

// AuthMiddleware creates JWT authentication middleware.
func AuthMiddleware(authService *services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				slog.Warn("auth: missing authorization header",
					"path", r.URL.Path,
					"remote", r.RemoteAddr,
				)
				writeError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				slog.Warn("auth: invalid authorization header format",
					"path", r.URL.Path,
				)
				writeError(w, http.StatusUnauthorized, "invalid authorization header format")
				return
			}

			userID, role, err := authService.ValidateToken(parts[1])
			if err != nil {
				slog.Warn("auth: token validation failed",
					"error", err,
					"path", r.URL.Path,
				)
				writeError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			slog.Debug("auth: token validated",
				"user_id", userID,
				"role", role,
				"path", r.URL.Path,
			)

			ctx := context.WithValue(r.Context(), ContextUserID, userID)
			ctx = context.WithValue(ctx, ContextRole, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns middleware that enforces a specific role.
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRole := GetRole(r.Context())
			if userRole != role {
				slog.Warn("auth: role mismatch",
					"required", role,
					"actual", userRole,
					"user_id", GetUserID(r.Context()),
					"path", r.URL.Path,
				)
				writeError(w, http.StatusForbidden, "access denied: requires "+role+" role")
				return
			}
			next.ServeHTTP(w, r.WithContext(r.Context()))
		})
	}
}

// GetUserID extracts the user ID from the request context.
func GetUserID(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(ContextUserID).(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}

// GetRole extracts the user role from the request context.
func GetRole(ctx context.Context) string {
	if role, ok := ctx.Value(ContextRole).(string); ok {
		return role
	}
	return ""
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
