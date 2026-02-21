package config

import "testing"

func TestLoadInvalidMaxConns(t *testing.T) {
	t.Setenv("JWT_HMAC_SECRET", "test-secret")
	t.Setenv("USER_DB_MAX_CONNS", "invalid")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid USER_DB_MAX_CONNS")
	}
}

func TestLoadInvalidDuration(t *testing.T) {
	t.Setenv("JWT_HMAC_SECRET", "test-secret")
	t.Setenv("ACCESS_TOKEN_TTL", "bad-duration")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid ACCESS_TOKEN_TTL")
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("JWT_HMAC_SECRET", "test-secret")
	t.Setenv("USER_SERVICE_GRPC_ADDR", "")
	t.Setenv("USER_DB_DSN", "")
	t.Setenv("USER_DB_MAX_CONNS", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("USER_DB_MIGRATIONS_PATH", "")
	t.Setenv("ACCESS_TOKEN_TTL", "")
	t.Setenv("REFRESH_TOKEN_TTL", "")
	t.Setenv("TOKEN_ISSUER", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.UserServiceGRPCAddr != ":50051" {
		t.Fatalf("expected default grpc addr :50051, got %q", cfg.UserServiceGRPCAddr)
	}
	if cfg.UserDBMaxConns != 10 {
		t.Fatalf("expected default max conns 10, got %d", cfg.UserDBMaxConns)
	}
	if cfg.AccessTokenTTL.String() != "15m0s" {
		t.Fatalf("expected default access token ttl 15m, got %s", cfg.AccessTokenTTL)
	}
	if cfg.RefreshTokenTTL.String() != "720h0m0s" {
		t.Fatalf("expected default refresh token ttl 720h, got %s", cfg.RefreshTokenTTL)
	}
}
