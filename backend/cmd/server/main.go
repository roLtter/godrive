package main

import (
	"context"
	"log"

	"cloudstore/backend/internal/config"
	_ "cloudstore/backend/internal/dbmigrate"
	"cloudstore/backend/internal/logger"
	"cloudstore/backend/internal/middleware"
	minioClient "cloudstore/backend/internal/storage/minio"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	zlog, err := logger.Init(cfg.AppEnv)
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	zlog.Info("config loaded",
		zap.String("app_env", cfg.AppEnv),
	)
	zlog.Info("migrations path", zap.String("path", cfg.MigrationsPath))

	if _, err := minioClient.New(
		context.Background(),
		cfg.MinIOURL,
		cfg.MinIORootUser,
		cfg.MinIORootPass,
		cfg.MinIOBucket,
		cfg.PresignTTLMin,
	); err != nil {
		zlog.Fatal("failed to init minio client", zap.Error(err))
	}
	zlog.Info("minio client initialized",
		zap.String("bucket", cfg.MinIOBucket),
		zap.Int("presign_ttl_min", cfg.PresignTTLMin),
	)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestLogger(zlog))
	router.GET("/health", func(c *gin.Context) {
		c.String(200, "OK")
	})

	zlog.Info("cloudstore server starting", zap.String("port", cfg.Port))
	if err := router.Run(":" + cfg.Port); err != nil {
		zlog.Fatal("failed to start server", zap.Error(err))
	}
}
