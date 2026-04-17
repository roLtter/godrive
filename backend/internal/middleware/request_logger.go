package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequestLogger logs core request metadata in structured format.
func RequestLogger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		log.Info("http_request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.String("latency", time.Since(start).String()),
			zap.String("client_ip", c.ClientIP()),
		)
	}
}
