package usergrpc

import (
	"context"
	"fmt"
	"net"

	usersv1 "github.com/ozankenangungor/go-commerce/api/gen/go/users/v1"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Server wraps the user service gRPC server.
type Server struct {
	addr         string
	logger       zerolog.Logger
	grpcServer   *grpc.Server
	healthServer *health.Server
}

// NewServer configures gRPC services and returns a server.
func NewServer(addr string, logger zerolog.Logger, userService usersv1.UserServiceServer) (*Server, error) {
	if addr == "" {
		return nil, fmt.Errorf("grpc address is required")
	}
	if userService == nil {
		return nil, fmt.Errorf("user service handler is required")
	}

	grpcServer := grpc.NewServer()
	healthServer := health.NewServer()

	usersv1.RegisterUserServiceServer(grpcServer, userService)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	reflection.Register(grpcServer)

	return &Server{
		addr:         addr,
		logger:       logger,
		grpcServer:   grpcServer,
		healthServer: healthServer,
	}, nil
}

// Start starts the gRPC listener.
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen grpc: %w", err)
	}

	s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	s.healthServer.SetServingStatus(usersv1.UserService_ServiceDesc.ServiceName, grpc_health_v1.HealthCheckResponse_SERVING)

	s.logger.Info().Str("addr", s.addr).Msg("user service grpc listening")

	if err := s.grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("serve grpc: %w", err)
	}
	return nil
}

// Shutdown gracefully stops the gRPC server, forcing stop if timeout is exceeded.
func (s *Server) Shutdown(ctx context.Context) error {
	s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	s.healthServer.SetServingStatus(usersv1.UserService_ServiceDesc.ServiceName, grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	done := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		s.logger.Warn().Msg("grpc graceful stop timeout reached, forcing stop")
		s.grpcServer.Stop()
		<-done
		return ctx.Err()
	}
}
