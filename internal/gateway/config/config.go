package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	defaultGatewayHTTPAddr     = ":8080"
	defaultUserServiceGRPCAddr = "localhost:50051"
	defaultGRPCDialTimeout     = 3 * time.Second
	defaultAuthRPCTimeout      = 2 * time.Second
	defaultLogLevel            = "info"
)

// Config contains runtime configuration for the API gateway.
type Config struct {
	GatewayHTTPAddr     string
	UserServiceGRPCAddr string
	GRPCDialTimeout     time.Duration
	AuthRPCTimeout      time.Duration
	LogLevel            string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (Config, error) {
	cfg := Config{
		GatewayHTTPAddr:     getEnv("GATEWAY_HTTP_ADDR", defaultGatewayHTTPAddr),
		UserServiceGRPCAddr: getEnv("USER_SERVICE_GRPC_ADDR", defaultUserServiceGRPCAddr),
		LogLevel:            strings.TrimSpace(getEnv("LOG_LEVEL", defaultLogLevel)),
	}

	var err error
	cfg.GRPCDialTimeout, err = getDurationEnv("GRPC_DIAL_TIMEOUT", defaultGRPCDialTimeout)
	if err != nil {
		return Config{}, err
	}

	cfg.AuthRPCTimeout, err = getDurationEnv("AUTH_RPC_TIMEOUT", defaultAuthRPCTimeout)
	if err != nil {
		return Config{}, err
	}

	if strings.TrimSpace(cfg.GatewayHTTPAddr) == "" {
		return Config{}, fmt.Errorf("GATEWAY_HTTP_ADDR cannot be empty")
	}
	if strings.TrimSpace(cfg.UserServiceGRPCAddr) == "" {
		return Config{}, fmt.Errorf("USER_SERVICE_GRPC_ADDR cannot be empty")
	}
	if cfg.GRPCDialTimeout <= 0 {
		return Config{}, fmt.Errorf("GRPC_DIAL_TIMEOUT must be > 0")
	}
	if cfg.AuthRPCTimeout <= 0 {
		return Config{}, fmt.Errorf("AUTH_RPC_TIMEOUT must be > 0")
	}
	if cfg.LogLevel == "" {
		return Config{}, fmt.Errorf("LOG_LEVEL cannot be empty")
	}

	return cfg, nil
}

func getDurationEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return duration, nil
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
