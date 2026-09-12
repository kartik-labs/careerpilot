// Package postgres provides the PostgreSQL storage abstraction for CareerPilot.
//
// Business logic must depend on interfaces defined by the storage callers,
// not on this package directly, so persistence can be swapped without
// touching domain code.
package postgres

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// DB wraps a PostgreSQL connection pool.
type DB struct {
	conn *sql.DB
}

// Open connects to PostgreSQL using the given DSN and verifies connectivity.
func Open(ctx context.Context, dsn string) (*DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("postgres: dsn must not be empty")
	}

	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: open: %w", err)
	}

	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}

	return &DB{conn: conn}, nil
}

// Close releases the underlying connection pool.
func (d *DB) Close() error {
	return d.conn.Close()
}

// Ping verifies the database connection is alive.
func (d *DB) Ping(ctx context.Context) error {
	return d.conn.PingContext(ctx)
}

// Conn exposes the underlying *sql.DB for repository implementations.
// Repositories should accept *sql.DB or *sql.Tx directly rather than this
// package's DB type, keeping persistence details out of domain interfaces.
func (d *DB) Conn() *sql.DB {
	return d.conn
}
