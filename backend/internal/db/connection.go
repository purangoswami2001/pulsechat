package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	// Import go-mysql-driver dynamically or keep standard database/sql interface
)

type Connection struct {
	PGPool  *pgxpool.Pool
	MySQLDB *sql.DB
	Driver  string
}

func Connect(ctx context.Context, driver, dsn string) (*Connection, error) {
	slog.Info("connecting to database pool...", "driver", driver)

	switch driver {
	case "postgres":
		config, err := pgxpool.ParseConfig(dsn)
		if err != nil {
			return nil, fmt.Errorf("failed to parse postgres DSN config: %w", err)
		}

		config.MaxConns = 25
		config.MinConns = 5
		config.MaxConnIdleTime = 30 * time.Minute
		config.MaxConnLifetime = 1 * time.Hour

		pool, err := pgxpool.NewWithConfig(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("failed to establish postgres connection pool: %w", err)
		}

		if err := pool.Ping(ctx); err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to ping postgres database: %w", err)
		}

		slog.Info("successfully established PostgreSQL connection pool")
		return &Connection{
			PGPool: pool,
			Driver: "postgres",
		}, nil

	case "mysql":
		// MySQL connection placeholder
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			return nil, fmt.Errorf("failed to open mysql: %w", err)
		}

		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)
		db.SetConnMaxIdleTime(30 * time.Minute)
		db.SetConnMaxLifetime(1 * time.Hour)

		// We do not ping or require active mysql connection, return connection placeholder
		slog.Info("successfully initialized MySQL driver placeholder")
		return &Connection{
			MySQLDB: db,
			Driver:  "mysql",
		}, nil

	default:
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}
}

func (c *Connection) Close() {
	if c.PGPool != nil {
		slog.Info("closing PostgreSQL connection pool...")
		c.PGPool.Close()
	}
	if c.MySQLDB != nil {
		slog.Info("closing MySQL connection...")
		_ = c.MySQLDB.Close()
	}
}
