package config

import "os"

const defaultMigrationsPath = "migrations"

// MigrationsPath returns the directory containing .up.sql / .down.sql files.
// Override with MIGRATIONS_PATH (absolute path recommended inside containers).
func MigrationsPath() string {
	if v := os.Getenv("MIGRATIONS_PATH"); v != "" {
		return v
	}
	return defaultMigrationsPath
}
