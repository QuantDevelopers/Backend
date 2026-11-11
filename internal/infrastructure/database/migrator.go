package database

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RunMigrations executes embedded *.up.sql migration files in lexical order idempotently.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migrationsFS fs.FS, logger *slog.Logger) error {
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}

	entries, err := fs.ReadDir(migrationsFS, ".")
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}

	var upFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			upFiles = append(upFiles, entry.Name())
		}
	}
	sort.Strings(upFiles)

	for _, filename := range upFiles {
		var alreadyApplied bool
		err := pool.QueryRow(
			ctx,
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)",
			filename,
		).Scan(&alreadyApplied)
		if err != nil {
			return fmt.Errorf("check migration status for %s: %w", filename, err)
		}
		if alreadyApplied {
			continue
		}

		sqlBytes, err := fs.ReadFile(migrationsFS, filename)
		if err != nil {
			return fmt.Errorf("read migration file %s: %w", filename, err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration transaction for %s: %w", filename, err)
		}

		if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("execute migration %s: %w", filename, err)
		}

		if _, err := tx.Exec(
			ctx,
			"INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT (version) DO NOTHING",
			filename,
		); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record migration %s: %w", filename, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", filename, err)
		}

		if logger != nil {
			logger.Info("applied database migration", "version", filename)
		}
	}

	return nil
}
