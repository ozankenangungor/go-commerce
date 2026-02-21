package db

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations applies SQL migrations from the provided directory.
func RunMigrations(dsn string, migrationsPath string) error {
	if dsn == "" {
		return fmt.Errorf("db dsn is required")
	}
	if migrationsPath == "" {
		return fmt.Errorf("migrations path is required")
	}

	absolutePath, err := filepath.Abs(migrationsPath)
	if err != nil {
		return fmt.Errorf("resolve migrations path: %w", err)
	}

	migrator, err := migrate.New("file://"+absolutePath, dsn)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	upErr := migrator.Up()
	if upErr != nil && !errors.Is(upErr, migrate.ErrNoChange) {
		sourceErr, dbErr := migrator.Close()
		if sourceErr != nil {
			return fmt.Errorf("close migrator source after migration error: %w", sourceErr)
		}
		if dbErr != nil {
			return fmt.Errorf("close migrator db after migration error: %w", dbErr)
		}
		return fmt.Errorf("run migrations: %w", upErr)
	}

	sourceErr, dbErr := migrator.Close()
	if sourceErr != nil {
		return fmt.Errorf("close migrator source: %w", sourceErr)
	}
	if dbErr != nil {
		return fmt.Errorf("close migrator db: %w", dbErr)
	}

	return nil
}
