package config

import (
	"fmt"

	"github.com/spf13/viper"
)

const (
	defaultMigrationsPath = "migrations"
	defaultPort           = "8080"
	defaultAppEnv         = "dev"
	defaultMinIOBucket    = "cloudstore"
	defaultPresignTTLMin  = 15
)

// Config is the single source of runtime environment settings.
type Config struct {
	DBURL          string
	RedisURL       string
	MinIOURL       string
	MinIORootUser  string
	MinIORootPass  string
	MinIOBucket    string
	PresignTTLMin  int
	Port           string
	AppEnv         string
	MigrationsPath string
}

// Load reads application configuration from environment and validates it.
func Load() (Config, error) {
	v := viper.New()
	v.SetEnvPrefix("")
	v.AutomaticEnv()

	v.SetDefault("PORT", defaultPort)
	v.SetDefault("APP_ENV", defaultAppEnv)
	v.SetDefault("MIGRATIONS_PATH", defaultMigrationsPath)
	v.SetDefault("MINIO_BUCKET", defaultMinIOBucket)
	v.SetDefault("MINIO_PRESIGN_TTL_MIN", defaultPresignTTLMin)

	cfg := Config{
		DBURL:          v.GetString("DB_URL"),
		RedisURL:       v.GetString("REDIS_URL"),
		MinIOURL:       v.GetString("MINIO_URL"),
		MinIORootUser:  v.GetString("MINIO_ROOT_USER"),
		MinIORootPass:  v.GetString("MINIO_ROOT_PASSWORD"),
		MinIOBucket:    v.GetString("MINIO_BUCKET"),
		PresignTTLMin:  v.GetInt("MINIO_PRESIGN_TTL_MIN"),
		Port:           v.GetString("PORT"),
		AppEnv:         v.GetString("APP_ENV"),
		MigrationsPath: v.GetString("MIGRATIONS_PATH"),
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
	if c.MinIOBucket == "" {
		return fmt.Errorf("config validation failed: MINIO_BUCKET is required")
	}
	if c.PresignTTLMin <= 0 {
		return fmt.Errorf("config validation failed: MINIO_PRESIGN_TTL_MIN must be > 0")
	}
	return nil
}
