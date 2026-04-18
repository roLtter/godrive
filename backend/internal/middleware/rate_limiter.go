package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	redisClient "cloudstore/backend/internal/cache/redis"
	"github.com/gin-gonic/gin"
	redisv9 "github.com/redis/go-redis/v9"
)

// RateLimiterConfig controls sliding-window limiter behavior.
type RateLimiterConfig struct {
	Limit    int
	Window   time.Duration
	KeyPrefix string
}

// RateLimiter applies Redis sliding-window request limiting.
func RateLimiter(redis *redisClient.Client, cfg RateLimiterConfig) gin.HandlerFunc {
	prefix := strings.TrimSpace(cfg.KeyPrefix)
	if prefix == "" {
		prefix = "ratelimit"
	}

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now().UTC()
		windowStart := now.Add(-cfg.Window).UnixMilli()
		nowMillis := now.UnixMilli()
		key := fmt.Sprintf("%s:%s:%s", prefix, clientIP, c.FullPath())
		if c.FullPath() == "" {
			key = fmt.Sprintf("%s:%s:%s", prefix, clientIP, c.Request.URL.Path)
		}

		member := strconv.FormatInt(now.UnixNano(), 10)

		pipe := redis.TxPipeline()
		pipe.ZRemRangeByScore(c.Request.Context(), key, "-inf", strconv.FormatInt(windowStart, 10))
		pipe.ZAdd(c.Request.Context(), key, redisv9.Z{
			Score:  float64(nowMillis),
			Member: member,
		})
		countCmd := pipe.ZCard(c.Request.Context(), key)
		pipe.Expire(c.Request.Context(), key, cfg.Window+time.Second)
		if _, err := pipe.Exec(c.Request.Context()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "rate limiter failure"})
			c.Abort()
			return
		}

		requests := int(countCmd.Val())
		remaining := cfg.Limit - requests
		if remaining < 0 {
			remaining = 0
		}
		c.Header("X-RateLimit-Limit", strconv.Itoa(cfg.Limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Window-Seconds", strconv.Itoa(int(cfg.Window.Seconds())))

		if requests > cfg.Limit {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			c.Abort()
			return
		}

		c.Next()
	}
}
