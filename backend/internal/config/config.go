package config

import (
	"fmt"
	"os"
)

const (
	defaultMigrationsPath = "migrations"
	defaultPort           = "8080"
)

// Config is the single source of runtime environment settings.
type Config struct {
	DBURL          string
	RedisURL       string
	MinIOURL       string
	MinIORootUser  string
	MinIORootPass  string
	Port           string
	MigrationsPath string
}

// Load reads application configuration from environment and validates it.
func Load() (Config, error) {
	cfg := Config{
		DBURL:          os.Getenv("DB_URL"),
		RedisURL:       os.Getenv("REDIS_URL"),
		MinIOURL:       os.Getenv("MINIO_URL"),
		MinIORootUser:  os.Getenv("MINIO_ROOT_USER"),
		MinIORootPass:  os.Getenv("MINIO_ROOT_PASSWORD"),
		Port:           getEnvOrDefault("PORT", defaultPort),
		MigrationsPath: getEnvOrDefault("MIGRATIONS_PATH", defaultMigrationsPath),
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Validate checks required configuration invariants.
func (c Config) Validate() error {
	if c.DBURL == "" {
		return fmt.Errorf("config validation failed: DB_URL is required")
	}
	if c.MinIORootUser == "" || c.MinIORootPass == "" {
		return fmt.Errorf("config validation failed: MINIO_ROOT_USER and MINIO_ROOT_PASSWORD are required")
	}
	return nil
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
