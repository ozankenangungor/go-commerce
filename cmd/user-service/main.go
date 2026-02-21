package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	userconfig "github.com/ozankenangungor/go-commerce/internal/user/config"
	userdb "github.com/ozankenangungor/go-commerce/internal/user/db"
	usergrpc "github.com/ozankenangungor/go-commerce/internal/user/grpc"
	userhandlers "github.com/ozankenangungor/go-commerce/internal/user/grpc/handlers"
	"github.com/rs/zerolog"
)

func main() {
	cfg, err := userconfig.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	logger, err := newLogger(cfg.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "configure logger: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbPool, err := userdb.NewPool(ctx, cfg.UserDBDSN, cfg.UserDBMaxConns)
	if err != nil {
		logger.Error().Err(err).Msg("failed to initialize db pool")
		os.Exit(1)
	}
	defer dbPool.Close()

	if err := userdb.RunMigrations(cfg.UserDBDSN, cfg.MigrationsPath); err != nil {
		logger.Error().Err(err).Msg("failed to run migrations")
		os.Exit(1)
	}

	handler := userhandlers.NewUserService(logger, dbPool)
	grpcServer, err := usergrpc.NewServer(cfg.UserServiceGRPCAddr, logger, handler)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create grpc server")
		os.Exit(1)
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- grpcServer.Start()
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalCh)

	select {
	case sig := <-signalCh:
		logger.Info().Str("signal", sig.String()).Msg("shutdown signal received")
	case err := <-serverErr:
		if err != nil {
			logger.Error().Err(err).Msg("grpc server exited unexpectedly")
			os.Exit(1)
		}
		return
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := grpcServer.Shutdown(shutdownCtx); err != nil && err != context.DeadlineExceeded {
		logger.Error().Err(err).Msg("grpc shutdown failed")
		os.Exit(1)
	}

	select {
	case err := <-serverErr:
		if err != nil {
			logger.Error().Err(err).Msg("grpc server exited with error")
			os.Exit(1)
		}
	case <-time.After(6 * time.Second):
		logger.Warn().Msg("timeout waiting for grpc server goroutine to exit")
	}
}

func newLogger(level string) (zerolog.Logger, error) {
	parsedLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		return zerolog.Logger{}, fmt.Errorf("parse LOG_LEVEL: %w", err)
	}

	logger := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Str("service", "user-service").
		Logger().
		Level(parsedLevel)

	return logger, nil
}
