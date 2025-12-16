package repository

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations runs database migrations
func RunMigrations(databaseURL string) error {
	m, err := migrate.New(
		"file://internal/repository/migrations",
		databaseURL,
	)
	if err != nil {
		return fmt.Errorf("create migration instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		// Handle dirty database state by forcing to the previous clean version
		if dirtyErr, ok := err.(migrate.ErrDirty); ok {
			version, dirty, verr := m.Version()
			if verr != nil {
				return fmt.Errorf("get current migration version: %w", verr)
			}

			if dirty {
				forceVersion := int(version) - 1
				if forceVersion < 0 {
					forceVersion = 0
				}

				if ferr := m.Force(forceVersion); ferr != nil {
					return fmt.Errorf("force clean migration version %d: %w", forceVersion, ferr)
				}

				// Retry migrations after cleaning dirty state
				if err := m.Up(); err != nil && err != migrate.ErrNoChange {
					return fmt.Errorf("rerun migrations after dirty state: %w", err)
				}

				return nil
			}

			return fmt.Errorf("dirty migrations at version %d and could not auto-fix", dirtyErr.Version)
		}

		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
