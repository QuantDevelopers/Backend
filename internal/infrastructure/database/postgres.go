package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/QuantDevelopers/Backend/internal/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
)

type slogTracer struct {
	logger *slog.Logger
}

func (t slogTracer) Log(_ context.Context, level tracelog.LogLevel, msg string, data map[string]any) {
	t.logger.Debug("pgx", "level", level.String(), "msg", msg, "data", data)
}

// NewPool creates and configures a PostgreSQL connection pool.
func NewPool(ctx context.Context, cfg config.Settings, logger *slog.Logger) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxPoolConns()
	poolCfg.MinConns = 1
	poolCfg.MaxConnLifetime = time.Hour
	poolCfg.MaxConnIdleTime = 30 * time.Minute
	poolCfg.HealthCheckPeriod = time.Minute

	if cfg.DBEcho && logger != nil {
		poolCfg.ConnConfig.Tracer = &tracelog.TraceLog{
			Logger:   slogTracer{logger: logger},
			LogLevel: tracelog.LogLevelDebug,
		}
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	return pool, nil
}

// CheckConnection executes SELECT 1 to verify live database connectivity.
func CheckConnection(ctx context.Context, pool *pgxpool.Pool) bool {
	if pool == nil {
		return false
	}
	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var result int
	err := pool.QueryRow(checkCtx, "SELECT 1").Scan(&result)
	return err == nil && result == 1
}

// WaitForConnection retries Ping briefly during container startup so transient boot races recover cleanly.
func WaitForConnection(ctx context.Context, pool *pgxpool.Pool, maxAttempts int, delay time.Duration) error {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err := pool.Ping(pingCtx)
		cancel()
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}
	return fmt.Errorf("database unreachable after %d attempts: %w", maxAttempts, lastErr)
}

// Ensure pgx.Tx satisfies common query usage at compile time if needed elsewhere.
var _ pgx.Tx = (pgx.Tx)(nil)
