package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	usersclient "github.com/ozankenangungor/go-commerce/internal/gateway/clients/users"
	"github.com/ozankenangungor/go-commerce/internal/gateway/config"
	gatewayhttp "github.com/ozankenangungor/go-commerce/internal/gateway/http"
	"github.com/rs/zerolog"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	logger, err := newLogger(cfg.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "configure logger: %v\n", err)
		os.Exit(1)
	}

	usersClient, err := usersclient.NewClient(context.Background(), cfg.UserServiceGRPCAddr, cfg.GRPCDialTimeout)
	if err != nil {
		logger.Error().Err(err).Msg("failed to initialize users grpc client")
		os.Exit(1)
	}
	defer func() {
		if closeErr := usersClient.Close(); closeErr != nil {
			logger.Error().Err(closeErr).Msg("failed to close users grpc client")
		}
	}()

	server := gatewayhttp.NewServer(cfg, gatewayhttp.Dependencies{
		Logger:         logger,
		TokenValidator: usersClient,
		AuthRPCTimeout: cfg.AuthRPCTimeout,
	})

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start()
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalCh)

	select {
	case sig := <-signalCh:
		logger.Info().Str("signal", sig.String()).Msg("shutdown signal received")
	case err := <-serverErr:
		if err != nil {
			logger.Error().Err(err).Msg("api gateway stopped unexpectedly")
			os.Exit(1)
		}
		return
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("graceful shutdown failed")
		os.Exit(1)
	}

	select {
	case err := <-serverErr:
		if err != nil {
			logger.Error().Err(err).Msg("api gateway exited with error")
			os.Exit(1)
		}
	case <-time.After(6 * time.Second):
		logger.Warn().Msg("timeout waiting for server goroutine to exit")
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
		Str("service", "api-gateway").
		Logger().
		Level(parsedLevel)

	return logger, nil
}
