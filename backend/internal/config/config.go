package config

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all server configurations.
type Config struct {
	Port               string
	Env                string
	DBDSN              string
	RedisURL           string
	JWTSecret          string
	JWTExpirationHours time.Duration
	PubSubDriver       string // "memory" or "redis"
}

// Load reads config from the environment.
// It optionally loads from a local `.env` file if it exists.
func Load() (*Config, error) {
	// Try loading from .env if present
	if err := loadDotEnv(".env"); err != nil {
		// Log warning, not an error since variables might be injected via environment/docker
		slog.Info("No .env file loaded or error reading .env file", "error", err)
	}

	port := getEnv("PORT", "8080")
	env := getEnv("ENV", "development")
	dbDSN := getEnv("DB_DSN", "postgres://postgres:postgres@localhost:5432/pulsechat?sslmode=disable")
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379/0")
	jwtSecret := getEnv("JWT_SECRET", "super_secret_jwt_key_should_be_long_and_random")

	jwtExpHoursStr := getEnv("JWT_EXPIRATION_HOURS", "24")
	jwtExpHours, err := strconv.Atoi(jwtExpHoursStr)
	if err != nil {
		jwtExpHours = 24
	}

	pubSubDriver := getEnv("PUBSUB_DRIVER", "memory")
	if pubSubDriver != "memory" && pubSubDriver != "redis" {
		return nil, fmt.Errorf("invalid PUBSUB_DRIVER value %q (supported: memory, redis)", pubSubDriver)
	}

	return &Config{
		Port:               port,
		Env:                env,
		DBDSN:              dbDSN,
		RedisURL:           redisURL,
		JWTSecret:          jwtSecret,
		JWTExpirationHours: time.Duration(jwtExpHours) * time.Hour,
		PubSubDriver:       pubSubDriver,
	}, nil
}

// getEnv gets an environment variable or falls back to a default value.
func getEnv(key, defaultValue string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultValue
}

// loadDotEnv parses a basic .env file and sets environment variables.
func loadDotEnv(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // ignore if file doesn't exist
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines or comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// Strip quotes if present
		if (strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"")) ||
			(strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'")) {
			if len(val) >= 2 {
				val = val[1 : len(val)-1]
			}
		}

		// Only set if not already set by system environment
		if _, ok := os.LookupEnv(key); !ok {
			if err := os.Setenv(key, val); err != nil {
				return fmt.Errorf("failed to set env var %s: %w", key, err)
			}
		}
	}

	return scanner.Err()
}
