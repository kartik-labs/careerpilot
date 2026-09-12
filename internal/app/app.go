// Package app wires together CareerPilot's dependencies (config, logging,
// storage, HTTP server) into a runnable application. This is the single
// place where concrete implementations are chosen for the interfaces used
// by domain and delivery code.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/kartik-labs/careerpilot/internal/config"
	"github.com/kartik-labs/careerpilot/internal/httpserver"
	"github.com/kartik-labs/careerpilot/internal/logging"
	"github.com/kartik-labs/careerpilot/internal/storage/postgres"
)

// App holds the wired dependencies for the CareerPilot service.
type App struct {
	Config config.Config
	Logger *slog.Logger
	DB     *postgres.DB
	Server *http.Server
}

// New wires dependencies from configuration. DB is nil when DatabaseURL is
// empty, which is expected in local development before Postgres is
// provisioned.
func New(ctx context.Context, cfg config.Config) (*App, error) {
	logger := logging.New(cfg.LogLevel)

	var db *postgres.DB
	if cfg.DatabaseURL != "" {
		var err error
		db, err = postgres.Open(ctx, cfg.DatabaseURL)
		if err != nil {
			return nil, fmt.Errorf("app: connect to postgres: %w", err)
		}
	}

	srv := httpserver.New(logger)

	return &App{
		Config: cfg,
		Logger: logger,
		DB:     db,
		Server: &http.Server{
			Addr:    cfg.HTTPAddr,
			Handler: srv.Handler(),
		},
	}, nil
}

// Close releases resources held by the application.
func (a *App) Close() error {
	if a.DB != nil {
		return a.DB.Close()
	}
	return nil
}
