package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/pulsechat/backend/internal/config"
	"github.com/pulsechat/backend/internal/migrate"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	args := os.Args[1:]
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	migrator := migrate.NewMigrator(cfg.DBDriver, cfg.DBDSN)

	cmd := args[0]
	switch cmd {
	case "up":
		if err := migrator.Up(); err != nil {
			log.Fatalf("Failed to run up migrations: %v", err)
		}
	case "down":
		if err := migrator.Down(); err != nil {
			log.Fatalf("Failed to run down migrations: %v", err)
		}
	case "steps":
		if len(args) < 2 {
			log.Fatalf("steps requires an integer argument (positive or negative)")
		}
		steps, err := strconv.Atoi(args[1])
		if err != nil {
			log.Fatalf("invalid steps argument: %v", err)
		}
		if err := migrator.Steps(steps); err != nil {
			log.Fatalf("Failed to run steps: %v", err)
		}
	case "version":
		v, dirty, err := migrator.Version()
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		fmt.Printf("Version: %d (dirty: %t)\n", v, dirty)
	case "force":
		if len(args) < 2 {
			log.Fatalf("force requires a version integer argument")
		}
		v, err := strconv.Atoi(args[1])
		if err != nil {
			log.Fatalf("invalid version argument: %v", err)
		}
		if err := migrator.Force(v); err != nil {
			log.Fatalf("Failed to force version: %v", err)
		}
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: go run ./cmd/migrate <command> [args]")
	fmt.Println("Commands:")
	fmt.Println("  up          Apply all migrations")
	fmt.Println("  down        Rollback all migrations")
	fmt.Println("  steps <N>   Run N migration steps (positive to apply, negative to rollback)")
	fmt.Println("  version     Print current migration version")
	fmt.Println("  force <V>   Force migration version V (useful if dirty state needs correction)")
}
