package builder

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// App represents the application with all its components
type App struct {
	server *http.Server
	db     *pgxpool.Pool
	logger *zap.Logger
}

// Run starts the application and all its daemons
func (a *App) Run() error {
	// Start HTTP server in goroutine
	errChan := make(chan error, 1)
	go func() {
		a.logger.Info("Starting HTTP server", zap.String("addr", a.server.Addr))
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for interrupt signal or server error
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errChan:
		a.logger.Error("Server error", zap.Error(err))
		return err
	case sig := <-sigChan:
		a.logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
	}

	// Graceful shutdown
	return a.shutdown()
}

// shutdown gracefully shuts down the application
func (a *App) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a.logger.Info("Shutting down server gracefully")

	if err := a.server.Shutdown(ctx); err != nil {
		a.logger.Error("Server shutdown error", zap.Error(err))
		return err
	}

	a.logger.Info("Closing database connections")
	if a.db != nil {
		a.db.Close()
	}

	a.logger.Info("Application stopped gracefully")
	return nil
}
