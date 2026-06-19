package db

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB represents a wrapper around the pgx connection pool and generated Queries.
type DB struct {
	Pool    *pgxpool.Pool
	Queries *Queries
}

// Connect initializes the database connection pool and runs auto-migrations.
func Connect(ctx context.Context, dsn string) (*DB, error) {
	slog.Info("connecting to PostgreSQL database pool...")

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSN config: %w", err)
	}

	// Adjust connections bounds
	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnIdleTime = 30 * time.Minute
	config.MaxConnLifetime = 1 * time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to establish connection pool: %w", err)
	}

	// Ping database to verify connection is open
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	slog.Info("successfully established database connection pool")

	// Run auto-migrations
	if err := runMigrations(dsn); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database migration failed: %w", err)
	}

	// Create wrapping instance with generated SQLC Queries
	queries := New(pool)

	return &DB{
		Pool:    pool,
		Queries: queries,
	}, nil
}

// Close safely shuts down the pgx pool connection.
func (db *DB) Close() {
	if db.Pool != nil {
		slog.Info("closing database connection pool...")
		db.Pool.Close()
	}
}

// runMigrations parses embedded SQL migrations and applies them to the database.
func runMigrations(dsn string) error {
	slog.Info("checking database schemas auto-migrations...")

	// Convert embedded FS into golang-migrate source driver
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to load embedded migrations driver: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, dsn)
	if err != nil {
		return fmt.Errorf("failed to initialize migration instance: %w", err)
	}
	defer m.Close()

	// Apply migrations
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("database schema is up to date (no changes applied)")
			return nil
		}
		return fmt.Errorf("failed to run migrations up: %w", err)
	}

	slog.Info("database migrations applied successfully")
	return nil
}
