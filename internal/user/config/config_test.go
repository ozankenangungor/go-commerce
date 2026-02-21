package config

import (
	"os"
	"testing"
)

func TestLoadInvalidMaxConns(t *testing.T) {
	t.Setenv("USER_DB_MAX_CONNS", "invalid")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid USER_DB_MAX_CONNS")
	}
}

func TestLoadDefaults(t *testing.T) {
	keys := []string{
		"USER_SERVICE_GRPC_ADDR",
		"USER_DB_DSN",
		"USER_DB_MAX_CONNS",
		"LOG_LEVEL",
		"USER_DB_MIGRATIONS_PATH",
	}

	for _, key := range keys {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
	}

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
}
