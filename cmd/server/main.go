package main

import (
	"context"
	"errors"
	"log/slog"
	stdhttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/QuantDevelopers/Backend/internal/config"
	"github.com/QuantDevelopers/Backend/internal/container"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
	httpiface "github.com/QuantDevelopers/Backend/internal/interfaces/http"
	"github.com/QuantDevelopers/Backend/migrations"
)

func main() {
	cfg := config.Load()

	logLevel := slog.LevelInfo
	if cfg.Debug {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := database.NewPool(ctx, cfg, logger)
	if err != nil {
		logger.Error("failed to initialize database pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := database.WaitForConnection(ctx, pool, 10, 2*time.Second); err != nil {
		logger.Error("database connection failed during startup", "error", err)
		os.Exit(1)
	}

	if err := database.RunMigrations(ctx, pool, migrations.FS, logger); err != nil {
		logger.Error("failed to run database migrations", "error", err)
		os.Exit(1)
	}

	appContainer := container.New(cfg, logger, pool)

	// Seed the 25 default underlying assets on startup (matching FastAPI lifespan behavior).
	if _, err := appContainer.UnderlyingService.ListUnderlyings(ctx); err != nil {
		logger.Error("failed to seed default underlyings", "error", err)
		os.Exit(1)
	}

	router := httpiface.NewRouter(cfg, logger, httpiface.RouterDeps{
		HealthChecker:     appContainer,
		UnderlyingUseCase: appContainer.UnderlyingService,
	})

	errCh := make(chan error, 1)
	go func() {
		logger.Info("starting QuantBackend Go/Echo server", "address", cfg.Address(), "app", cfg.AppName)
		if startErr := router.Start(cfg.Address()); startErr != nil && !errors.Is(startErr, stdhttp.ErrServerClosed) {
			errCh <- startErr
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received, draining server")
	case serveErr := <-errCh:
		logger.Error("HTTP server stopped unexpectedly", "error", serveErr)
		os.Exit(1)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := router.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped cleanly")
}
