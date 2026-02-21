package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	usersclient "github.com/ozankenangungor/go-commerce/internal/gateway/clients/users"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeTokenValidator struct {
	validateFunc func(ctx context.Context, accessToken string, requestID string) (string, []string, error)
}

func (f fakeTokenValidator) ValidateAccessToken(ctx context.Context, accessToken string, requestID string) (string, []string, error) {
	if f.validateFunc == nil {
		return "", nil, errors.New("validate function not set")
	}
	return f.validateFunc(ctx, accessToken, requestID)
}

func TestAuthMissingAuthorization(t *testing.T) {
	called := false
	handler := newProtectedHandler(t, fakeTokenValidator{
		validateFunc: func(ctx context.Context, accessToken string, requestID string) (string, []string, error) {
			called = true
			return "", nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
	if called {
		t.Fatal("validator should not be called when Authorization header is missing")
	}
	assertErrorBody(t, rr, "unauthorized")
}

func TestAuthMalformedAuthorizationHeader(t *testing.T) {
	called := false
	handler := newProtectedHandler(t, fakeTokenValidator{
		validateFunc: func(ctx context.Context, accessToken string, requestID string) (string, []string, error) {
			called = true
			return "", nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	req.Header.Set("Authorization", "Token abc")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
	if called {
		t.Fatal("validator should not be called when Authorization header is malformed")
	}
	assertErrorBody(t, rr, "unauthorized")
}

func TestAuthInvalidTokenError(t *testing.T) {
	handler := newProtectedHandler(t, fakeTokenValidator{
		validateFunc: func(ctx context.Context, accessToken string, requestID string) (string, []string, error) {
			return "", nil, &usersclient.ValidateAccessTokenError{
				ErrCode:    "AUTH_INVALID_TOKEN",
				ErrMessage: "invalid token",
			}
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
	assertErrorBody(t, rr, "unauthorized")
}

func TestAuthUnavailableReturns503(t *testing.T) {
	handler := newProtectedHandler(t, fakeTokenValidator{
		validateFunc: func(ctx context.Context, accessToken string, requestID string) (string, []string, error) {
			return "", nil, status.Error(codes.Unavailable, "connection refused")
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	req.Header.Set("Authorization", "Bearer some-token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rr.Code)
	}
	assertErrorBody(t, rr, "auth_unavailable")
}

func TestAuthSuccessPassesContextValues(t *testing.T) {
	var capturedToken string
	var capturedRequestID string

	handler := newProtectedHandler(t, fakeTokenValidator{
		validateFunc: func(ctx context.Context, accessToken string, requestID string) (string, []string, error) {
			capturedToken = accessToken
			capturedRequestID = requestID
			return "user-123", []string{"customer", "premium"}, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if capturedToken != "valid-token" {
		t.Fatalf("expected token valid-token, got %q", capturedToken)
	}
	if capturedRequestID == "" {
		t.Fatal("expected non-empty request id passed to validator")
	}

	var body struct {
		UserID string   `json:"user_id"`
		Roles  []string `json:"roles"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body.UserID != "user-123" {
		t.Fatalf("expected user_id user-123, got %q", body.UserID)
	}
	if len(body.Roles) != 2 || body.Roles[0] != "customer" || body.Roles[1] != "premium" {
		t.Fatalf("unexpected roles: %#v", body.Roles)
	}
}

func newProtectedHandler(t *testing.T, validator TokenValidator) http.Handler {
	t.Helper()

	router := chi.NewRouter()
	router.Use(RequestID)
	router.Use(Auth(validator, time.Second))
	router.Get("/v1/me", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserIDFromContext(r.Context())
		if !ok {
			t.Fatalf("expected user id in context")
		}
		roles, ok := RolesFromContext(r.Context())
		if !ok {
			t.Fatalf("expected roles in context")
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"user_id": userID,
			"roles":   roles,
		})
	})

	return router
}

func assertErrorBody(t *testing.T, rr *httptest.ResponseRecorder, want string) {
	t.Helper()

	var payload map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal error body: %v", err)
	}

	if payload["error"] != want {
		t.Fatalf("expected error %q, got %q", want, payload["error"])
	}
}
