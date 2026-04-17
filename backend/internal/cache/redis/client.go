package redis

import (
	"context"
	"fmt"
	"time"

	redisv9 "github.com/redis/go-redis/v9"
)

// Config describes Redis connection and pool configuration.
type Config struct {
	URL          string
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
	MinIdleConns int
}

// Client wraps go-redis client.
type Client struct {
	*redisv9.Client
}

// New creates a Redis client and verifies connectivity.
func New(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("redis client: redis url is required")
	}

	opts, err := redisv9.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("redis client: parse url: %w", err)
	}

	opts.DialTimeout = cfg.DialTimeout
	opts.ReadTimeout = cfg.ReadTimeout
	opts.WriteTimeout = cfg.WriteTimeout
	opts.PoolSize = cfg.PoolSize
	opts.MinIdleConns = cfg.MinIdleConns

	rdb := redisv9.NewClient(opts)
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("redis client: ping failed: %w", err)
	}

	return &Client{Client: rdb}, nil
}

// HealthCheck verifies Redis connectivity.
func (c *Client) HealthCheck(ctx context.Context) error {
	if err := c.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis client health check failed: %w", err)
	}
	return nil
}
