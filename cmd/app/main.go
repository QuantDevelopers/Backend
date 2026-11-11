package main

import (
	"context"
	"errors"
	"flag"
	stdhttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/QuantDevelopers/Backend/bootstrap"
	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	domainlogger "github.com/QuantDevelopers/Backend/internal/domain/logger"
	"github.com/QuantDevelopers/Backend/internal/presentation/routes"
	"github.com/QuantDevelopers/Backend/wire"
	"github.com/labstack/echo/v4"
)

func main() {
	seedOnly := flag.Bool("seed-only", false, "run database AutoMigrate and seeders, then exit")
	flag.Parse()

	config := bootstrap.Run()

	app, err := wire.InitializeApplication(config)
	if err != nil {
		panic(err)
	}
	defer app.Logger.Close()

	if err := app.Database.DB.GetDB().AutoMigrate(
		&entity.UnderlyingAsset{},
		&entity.OptionContract{},
		&entity.DailyHistory{},
		&entity.TickRaw{},
		&entity.TickClean{},
		&entity.DataQualityLog{},
	); err != nil {
		app.Logger.Fatal("failed to auto-migrate database schema", domainlogger.Error("error", err))
	}

	app.Seeds.UnderlyingSeeder.SeedDefaultUnderlyings()
	app.Logger.Info("database auto-migration and default underlying seeding completed")

	if *seedOnly {
		return
	}

	echoEngine := echo.New()
	echoEngine.HideBanner = true
	echoEngine.HidePort = true

	routes.Run(echoEngine, app.ToAppRouter())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		app.Logger.Info(
			"starting QuantBackend Echo server",
			domainlogger.String("address", config.Env.Server.Address()),
			domainlogger.String("app", config.Env.Server.AppName),
		)
		if startErr := echoEngine.Start(config.Env.Server.Address()); startErr != nil && !errors.Is(startErr, stdhttp.ErrServerClosed) {
			errCh <- startErr
		}
	}()

	select {
	case <-ctx.Done():
		app.Logger.Info("shutdown signal received, draining server")
	case serveErr := <-errCh:
		app.Logger.Fatal("HTTP server stopped unexpectedly", domainlogger.Error("error", serveErr))
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := echoEngine.Shutdown(shutdownCtx); err != nil {
		app.Logger.Error("graceful shutdown failed", domainlogger.Error("error", err))
		os.Exit(1)
	}

	app.Logger.Info("server stopped cleanly")
}
