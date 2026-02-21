package gatewayhttp

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/ozankenangungor/go-commerce/internal/gateway/config"
	gatewaymiddleware "github.com/ozankenangungor/go-commerce/internal/gateway/http/middleware"
	"github.com/rs/zerolog"
)

// Dependencies holds constructor dependencies for the gateway HTTP server.
type Dependencies struct {
	Logger         zerolog.Logger
	TokenValidator gatewaymiddleware.TokenValidator
	AuthRPCTimeout time.Duration
}

// Server encapsulates the API gateway HTTP server.
type Server struct {
	httpServer *http.Server
	logger     zerolog.Logger
	ready      atomic.Bool
}

// NewServer builds a new API gateway HTTP server.
func NewServer(cfg config.Config, deps Dependencies) *Server {
	srv := &Server{
		logger: deps.Logger,
	}

	router := NewRouter(deps.Logger, deps.TokenValidator, deps.AuthRPCTimeout, srv.Ready)
	srv.httpServer = &http.Server{
		Addr:              cfg.GatewayHTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return srv
}

// Start starts listening for HTTP requests.
func (s *Server) Start() error {
	s.ready.Store(true)
	s.logger.Info().Str("addr", s.httpServer.Addr).Msg("api gateway listening")

	err := s.httpServer.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.ready.Store(false)
	return s.httpServer.Shutdown(ctx)
}

// Ready returns readiness state.
func (s *Server) Ready() bool {
	return s.ready.Load()
}
