package auth

import (
	"context"
	"testing"
	"time"

	redisClient "cloudstore/backend/internal/cache/redis"
	"github.com/alicebob/miniredis/v2"
)

func TestRefreshStore_SaveRotateInvalidate(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	ctx := context.Background()
	rdb, err := redisClient.New(ctx, redisClient.Config{
		URL:          "redis://" + mr.Addr() + "/0",
		DialTimeout:  time.Second,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
		PoolSize:     5,
		MinIdleConns: 1,
	})
	if err != nil {
		t.Fatalf("failed to create redis client: %v", err)
	}
	defer func() { _ = rdb.Close() }()

	store := NewRefreshStore(rdb)
	oldToken := "old-refresh-token"
	newToken := "new-refresh-token"

	if err := store.Save(ctx, oldToken, "user-1", time.Hour); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	userID, err := store.Rotate(ctx, oldToken, newToken, time.Hour)
	if err != nil {
		t.Fatalf("Rotate returned error: %v", err)
	}
	if userID != "user-1" {
		t.Fatalf("expected user-1, got %s", userID)
	}

	if err := store.Invalidate(ctx, newToken); err != nil {
		t.Fatalf("Invalidate returned error: %v", err)
	}
	if err := store.Invalidate(ctx, newToken); err == nil {
		t.Fatalf("expected error on second invalidate for missing token")
	}
}
