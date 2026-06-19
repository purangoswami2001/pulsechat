package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: go run cmd/migrate/main.go [up|down]")
		os.Exit(1)
	}

	cmd := args[0]

	// Get DSN from environment or fallback
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/pulsechat?sslmode=disable"
	}

	// Migrations are stored in internal/db/migrations
	migrationsPath := filepath.Join("internal", "db", "migrations")
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		// Fallback if run from parent directory
		migrationsPath = filepath.Join("backend", "internal", "db", "migrations")
	}

	m, err := migrate.New("file://"+migrationsPath, dsn)
	if err != nil {
		log.Fatalf("Failed to initialize migration instance: %v", err)
	}
	defer m.Close()

	switch cmd {
	case "up":
		fmt.Println("Running database migrations (up)...")
		if err := m.Up(); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				fmt.Println("Database schema is already up to date.")
				return
			}
			log.Fatalf("Failed to apply migrations: %v", err)
		}
		fmt.Println("Migrations applied successfully!")
	case "down":
		fmt.Println("Running database migrations (down)...")
		if err := m.Down(); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				fmt.Println("No migration change detected.")
				return
			}
			log.Fatalf("Failed to rollback migrations: %v", err)
		}
		fmt.Println("Migrations rolled back successfully!")
	default:
		fmt.Printf("Unknown command %q. Use 'up' or 'down'.\n", cmd)
		os.Exit(1)
	}
}
