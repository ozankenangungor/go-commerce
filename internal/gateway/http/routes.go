package gatewayhttp

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	gatewaymiddleware "github.com/ozankenangungor/go-commerce/internal/gateway/http/middleware"
	"github.com/rs/zerolog"
)

// NewRouter creates gateway HTTP routes and middleware stack.
func NewRouter(
	logger zerolog.Logger,
	validator gatewaymiddleware.TokenValidator,
	authRPCTimeout time.Duration,
	readyFn func() bool,
) http.Handler {
	if readyFn == nil {
		readyFn = func() bool { return false }
	}

	router := chi.NewRouter()
	router.Use(gatewaymiddleware.RequestID)
	router.Use(chimiddleware.Recoverer)
	router.Use(RequestLogger(logger))

	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	router.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if !readyFn() {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})

	router.Route("/v1", func(r chi.Router) {
		r.With(gatewaymiddleware.Auth(validator, authRPCTimeout)).Get("/me", func(w http.ResponseWriter, r *http.Request) {
			userID, ok := gatewaymiddleware.UserIDFromContext(r.Context())
			if !ok {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}

			roles, ok := gatewaymiddleware.RolesFromContext(r.Context())
			if !ok {
				roles = []string{}
			}

			writeJSON(w, http.StatusOK, map[string]any{
				"user_id": userID,
				"roles":   roles,
			})
		})
	})

	return router
}

// RequestLogger logs HTTP requests with structured fields.
func RequestLogger(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(wrapped, r)

			status := wrapped.Status()
			if status == 0 {
				status = http.StatusOK
			}

			logger.Info().
				Str("request_id", gatewaymiddleware.RequestIDFromContext(r.Context())).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", status).
				Int("bytes", wrapped.BytesWritten()).
				Dur("duration", time.Since(start)).
				Msg("http_request")
		})
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
