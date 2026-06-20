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

type Config struct {
	Port               string
	Env                string
	DBDriver           string // "postgres", "mysql"
	DBDSN              string
	AutoMigrate        bool
	RedisURL           string
	JWTSecret          string
	JWTExpirationHours time.Duration
	PubSubDriver       string // "memory" or "redis"
}

func Load() (*Config, error) {
	if err := loadDotEnv(".env"); err != nil {
		slog.Info("No .env file loaded or error reading .env file", "error", err)
	}

	port := getEnv("PORT", "8080")
	env := getEnv("ENV", "development")
	dbDriver := getEnv("DB_DRIVER", "postgres")
	dbDSN := getEnv("DB_DSN", "postgres://postgres:postgres@localhost:5432/pulsechat?sslmode=disable")
	autoMigrateStr := getEnv("AUTO_MIGRATE", "false")
	autoMigrate := autoMigrateStr == "true"
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
		DBDriver:           dbDriver,
		DBDSN:              dbDSN,
		AutoMigrate:        autoMigrate,
		RedisURL:           redisURL,
		JWTSecret:          jwtSecret,
		JWTExpirationHours: time.Duration(jwtExpHours) * time.Hour,
		PubSubDriver:       pubSubDriver,
	}, nil
}

func getEnv(key, defaultValue string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultValue
}

func loadDotEnv(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		if (strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"")) ||
			(strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'")) {
			if len(val) >= 2 {
				val = val[1 : len(val)-1]
			}
		}

		if _, ok := os.LookupEnv(key); !ok {
			if err := os.Setenv(key, val); err != nil {
				return fmt.Errorf("failed to set env var %s: %w", key, err)
			}
		}
	}

	return scanner.Err()
}
