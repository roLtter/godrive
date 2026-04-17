package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// Config describes sql.DB pool parameters.
type Config struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// Client wraps sql.DB.
type Client struct {
	*sql.DB
}

// New creates PostgreSQL client and verifies connectivity.
func New(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("postgres client: db url is required")
	}

	db, err := sql.Open("postgres", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("postgres client: open: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("postgres client: ping failed: %w", err)
	}

	return &Client{DB: db}, nil
}

// HealthCheck verifies database connectivity.
func (c *Client) HealthCheck(ctx context.Context) error {
	if err := c.PingContext(ctx); err != nil {
		return fmt.Errorf("postgres client health check failed: %w", err)
	}
	return nil
}
