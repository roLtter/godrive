package auth

import (
	"context"
	"fmt"
	"time"

	redisClient "cloudstore/backend/internal/cache/redis"
	redisv9 "github.com/redis/go-redis/v9"
)

// RefreshStore persists refresh tokens in Redis.
type RefreshStore struct {
	redis *redisClient.Client
}

// NewRefreshStore creates refresh store.
func NewRefreshStore(redis *redisClient.Client) *RefreshStore {
	return &RefreshStore{redis: redis}
}

func (s *RefreshStore) key(token string) string {
	return "auth:refresh:" + token
}

// Save stores a refresh token with TTL.
func (s *RefreshStore) Save(ctx context.Context, token, userID string, ttl time.Duration) error {
	if err := s.redis.Set(ctx, s.key(token), userID, ttl).Err(); err != nil {
		return fmt.Errorf("save refresh token: %w", err)
	}
	return nil
}

// Rotate validates old token and replaces it with a new one.
func (s *RefreshStore) Rotate(ctx context.Context, oldToken, newToken string, ttl time.Duration) (string, error) {
	userID, err := s.redis.Get(ctx, s.key(oldToken)).Result()
	if err != nil {
		if err == redisv9.Nil {
			return "", fmt.Errorf("refresh token not found")
		}
		return "", fmt.Errorf("refresh token not found")
	}

	if err := s.redis.Del(ctx, s.key(oldToken)).Err(); err != nil {
		return "", fmt.Errorf("delete old refresh token: %w", err)
	}
	if err := s.redis.Set(ctx, s.key(newToken), userID, ttl).Err(); err != nil {
		return "", fmt.Errorf("save new refresh token: %w", err)
	}

	return userID, nil
}

// Invalidate removes refresh token from Redis.
func (s *RefreshStore) Invalidate(ctx context.Context, token string) error {
	res, err := s.redis.Del(ctx, s.key(token)).Result()
	if err != nil {
		return fmt.Errorf("invalidate refresh token: %w", err)
	}
	if res == 0 {
		return fmt.Errorf("refresh token not found")
	}
	return nil
}
