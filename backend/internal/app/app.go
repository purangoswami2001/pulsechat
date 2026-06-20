package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pulsechat/backend/internal/logger"
	"github.com/pulsechat/backend/internal/migrate"
)

type App struct {
	deps *Dependencies
}

func NewApp(deps *Dependencies) *App {
	return &App{deps: deps}
}

func (a *App) Start() error {
	logger.Init(a.deps.Config.Env)
	slog.Info("starting PulseChat backend application")

	// 1. Auto-Migrations Check
	if a.deps.Config.AutoMigrate {
		slog.Info("AUTO_MIGRATE=true, applying migrations...")
		migrator := migrate.NewMigrator(a.deps.Config.DBDriver, a.deps.Config.DBDSN)
		if err := migrator.RunAutoMigrate(); err != nil {
			return fmt.Errorf("failed to apply migrations: %w", err)
		}
	} else {
		slog.Info("AUTO_MIGRATE=false, skipping auto migrations (CLI manually triggered only)")
	}

	// Ensure uploads dir
	if err := os.MkdirAll("./uploads", 0755); err != nil {
		return fmt.Errorf("failed to create uploads dir: %w", err)
	}

	// 2. Start WebSocket Hub in Background
	hubCtx, cancelHub := context.WithCancel(context.Background())
	defer cancelHub()
	go a.deps.WSHub.Run(hubCtx)

	// 3. Configure HTTP Server
	serverAddr := fmt.Sprintf(":%s", a.deps.Config.Port)
	server := &http.Server{
		Addr:         serverAddr,
		Handler:      a.deps.HTTPHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	shutdownError := make(chan error, 1)

	go func() {
		slog.Info("HTTP server listening", "addr", serverAddr, "env", a.deps.Config.Env)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			shutdownError <- err
		}
	}()

	// 4. Graceful Shutdown Signal Interception
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		slog.Info("received shutdown signal", "signal", sig.String())
	case err := <-shutdownError:
		slog.Error("server boot error", "error", err)
		return err
	}

	cancelHub()

	// Shutdown with context timeout
	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	slog.Info("shutting down HTTP server gracefully...")
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful HTTP shutdown failed", "error", err)
		return err
	}

	slog.Info("PulseChat backend stopped gracefully")
	return nil
}
