package sqlite

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"
	"time"
)

// RunMigrations aplica todas las migraciones *.up.sql pendientes desde migrationsFS.
// Las versiones aplicadas se registran en schema_migrations para garantizar idempotencia.
func RunMigrations(db *sql.DB, migrationsFS fs.FS) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT    PRIMARY KEY,
			applied_at INTEGER NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("creating schema_migrations: %w", err)
	}

	entries, err := fs.Glob(migrationsFS, "*.up.sql")
	if err != nil {
		return fmt.Errorf("listing migrations: %w", err)
	}
	sort.Strings(entries)

	for _, name := range entries {
		if err := applyMigration(db, migrationsFS, name); err != nil {
			return err
		}
	}

	return nil
}

func applyMigration(db *sql.DB, migrationsFS fs.FS, name string) error {
	version := strings.TrimSuffix(name, ".up.sql")

	var count int
	if err := db.QueryRow(
		`SELECT COUNT(1) FROM schema_migrations WHERE version = ?`, version,
	).Scan(&count); err != nil {
		return fmt.Errorf("checking migration %s: %w", version, err)
	}
	if count > 0 {
		return nil // ya aplicada
	}

	content, err := fs.ReadFile(migrationsFS, name)
	if err != nil {
		return fmt.Errorf("reading migration %s: %w", version, err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("beginning tx for migration %s: %w", version, err)
	}
	defer tx.Rollback() //nolint:errcheck // noop tras Commit

	if _, err := tx.Exec(string(content)); err != nil {
		return fmt.Errorf("executing migration %s: %w", version, err)
	}
	if _, err := tx.Exec(
		`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
		version, time.Now().Unix(),
	); err != nil {
		return fmt.Errorf("recording migration %s: %w", version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing migration %s: %w", version, err)
	}

	slog.Info("migración aplicada", "version", version)
	return nil
}
