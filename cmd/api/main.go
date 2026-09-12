// Command api runs the CareerPilot HTTP API server.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kartik-labs/careerpilot/internal/app"
	"github.com/kartik-labs/careerpilot/internal/config"
)

const shutdownTimeout = 10 * time.Second

func main() {
	if err := run(); err != nil {
		slog.Error("careerpilot exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	application, err := app.New(ctx, cfg)
	if err != nil {
		return err
	}
	defer application.Close()

	application.Logger.Info("starting careerpilot api", "addr", cfg.HTTPAddr, "env", cfg.Env)

	errCh := make(chan error, 1)
	go func() {
		if err := application.Server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		application.Logger.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return application.Server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
