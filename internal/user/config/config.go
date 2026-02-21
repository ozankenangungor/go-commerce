package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultUserServiceGRPCAddr = ":50051"
	defaultUserDBDSN           = "postgres://postgres:postgres@localhost:5433/user_service?sslmode=disable"
	defaultUserDBMaxConns      = 10
	defaultLogLevel            = "info"
	defaultMigrationsPath      = "internal/user/db/migrations"
	defaultAccessTokenTTL      = 15 * time.Minute
	defaultRefreshTokenTTL     = 720 * time.Hour
	defaultTokenIssuer         = "go-commerce"
)

// Config contains runtime configuration for user service.
type Config struct {
	UserServiceGRPCAddr string
	UserDBDSN           string
	UserDBMaxConns      int32
	LogLevel            string
	MigrationsPath      string
	JWTHMACSecret       string
	AccessTokenTTL      time.Duration
	RefreshTokenTTL     time.Duration
	TokenIssuer         string
}

// Load reads config from environment variables.
func Load() (Config, error) {
	cfg := Config{
		UserServiceGRPCAddr: getEnv("USER_SERVICE_GRPC_ADDR", defaultUserServiceGRPCAddr),
		UserDBDSN:           getEnv("USER_DB_DSN", defaultUserDBDSN),
		LogLevel:            getEnv("LOG_LEVEL", defaultLogLevel),
		MigrationsPath:      getEnv("USER_DB_MIGRATIONS_PATH", defaultMigrationsPath),
		JWTHMACSecret:       getEnv("JWT_HMAC_SECRET", ""),
		TokenIssuer:         getEnv("TOKEN_ISSUER", defaultTokenIssuer),
	}

	maxConns, err := getIntEnv("USER_DB_MAX_CONNS", defaultUserDBMaxConns)
	if err != nil {
		return Config{}, err
	}
	cfg.UserDBMaxConns = int32(maxConns)

	cfg.AccessTokenTTL, err = getDurationEnv("ACCESS_TOKEN_TTL", defaultAccessTokenTTL)
	if err != nil {
		return Config{}, err
	}

	cfg.RefreshTokenTTL, err = getDurationEnv("REFRESH_TOKEN_TTL", defaultRefreshTokenTTL)
	if err != nil {
		return Config{}, err
	}

	if cfg.UserServiceGRPCAddr == "" {
		return Config{}, fmt.Errorf("USER_SERVICE_GRPC_ADDR cannot be empty")
	}
	if cfg.UserDBDSN == "" {
		return Config{}, fmt.Errorf("USER_DB_DSN cannot be empty")
	}
	if cfg.UserDBMaxConns <= 0 {
		return Config{}, fmt.Errorf("USER_DB_MAX_CONNS must be > 0")
	}
	if cfg.LogLevel == "" {
		return Config{}, fmt.Errorf("LOG_LEVEL cannot be empty")
	}
	if cfg.MigrationsPath == "" {
		return Config{}, fmt.Errorf("USER_DB_MIGRATIONS_PATH cannot be empty")
	}
	if strings.TrimSpace(cfg.JWTHMACSecret) == "" {
		return Config{}, fmt.Errorf("JWT_HMAC_SECRET cannot be empty")
	}
	if cfg.AccessTokenTTL <= 0 {
		return Config{}, fmt.Errorf("ACCESS_TOKEN_TTL must be > 0")
	}
	if cfg.RefreshTokenTTL <= 0 {
		return Config{}, fmt.Errorf("REFRESH_TOKEN_TTL must be > 0")
	}
	if strings.TrimSpace(cfg.TokenIssuer) == "" {
		return Config{}, fmt.Errorf("TOKEN_ISSUER cannot be empty")
	}

	return cfg, nil
}

func getIntEnv(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}

func getDurationEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
