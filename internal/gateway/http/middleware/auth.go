package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type userIDContextKey struct{}
type rolesContextKey struct{}

type codedError interface {
	error
	Code() string
}

// TokenValidator validates bearer tokens against the user service.
type TokenValidator interface {
	ValidateAccessToken(ctx context.Context, accessToken string, requestID string) (userID string, roles []string, err error)
}

// Auth enforces bearer auth for protected routes.
func Auth(validator TokenValidator, authRPCTimeout time.Duration) func(http.Handler) http.Handler {
	if validator == nil {
		panic("token validator cannot be nil")
	}
	if authRPCTimeout <= 0 {
		panic("auth rpc timeout must be > 0")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := extractBearerToken(r.Header.Get("Authorization"))
			if !ok {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}

			requestID := RequestIDFromContext(r.Context())
			rpcCtx, cancel := context.WithTimeout(r.Context(), authRPCTimeout)
			defer cancel()

			userID, roles, err := validator.ValidateAccessToken(rpcCtx, token, requestID)
			if err != nil {
				if isInvalidTokenError(err) {
					writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
					return
				}
				if isUnavailableError(err) {
					writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "auth_unavailable"})
					return
				}

				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey{}, userID)
			ctx = context.WithValue(ctx, rolesContextKey{}, append([]string(nil), roles...))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext returns an authenticated user id from context.
func UserIDFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	userID, ok := ctx.Value(userIDContextKey{}).(string)
	if !ok || userID == "" {
		return "", false
	}
	return userID, true
}

// RolesFromContext returns authenticated roles from context.
func RolesFromContext(ctx context.Context) ([]string, bool) {
	if ctx == nil {
		return nil, false
	}
	roles, ok := ctx.Value(rolesContextKey{}).([]string)
	if !ok {
		return nil, false
	}
	return append([]string(nil), roles...), true
}

func extractBearerToken(headerValue string) (string, bool) {
	parts := strings.Fields(headerValue)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func isInvalidTokenError(err error) bool {
	var codeErr codedError
	if !errors.As(err, &codeErr) {
		return false
	}
	return strings.HasPrefix(codeErr.Code(), "AUTH_INVALID_")
}

func isUnavailableError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}

	grpcStatus, ok := status.FromError(err)
	if !ok {
		return false
	}

	switch grpcStatus.Code() {
	case codes.Unavailable, codes.DeadlineExceeded:
		return true
	default:
		return false
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if _, writeErr := w.Write(body); writeErr != nil {
		return
	}
}
