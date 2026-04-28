package main

import (
	"context"
	"log"
	"strings"
	"time"

	"cloudstore/backend/internal/auth"
	redisClient "cloudstore/backend/internal/cache/redis"
	"cloudstore/backend/internal/config"
	postgresClient "cloudstore/backend/internal/db/postgres"
	_ "cloudstore/backend/internal/dbmigrate"
	"cloudstore/backend/internal/files"
	"cloudstore/backend/internal/folders"
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

	db, err := postgresClient.New(context.Background(), postgresClient.Config{
		URL:             cfg.DBURL,
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 30 * time.Minute,
	})
	if err != nil {
		zlog.Fatal("failed to init postgres client", zap.Error(err))
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			zlog.Warn("failed to close postgres client", zap.Error(cerr))
		}
	}()
	if err := db.HealthCheck(context.Background()); err != nil {
		zlog.Fatal("postgres health check failed", zap.Error(err))
	}
	zlog.Info("postgres client initialized")

	redisCfg := redisClient.Config{
		URL:          cfg.RedisURL,
		DialTimeout:  time.Duration(cfg.RedisTimeoutMS) * time.Millisecond,
		ReadTimeout:  time.Duration(cfg.RedisTimeoutMS) * time.Millisecond,
		WriteTimeout: time.Duration(cfg.RedisTimeoutMS) * time.Millisecond,
		PoolSize:     cfg.RedisPoolSize,
		MinIdleConns: cfg.RedisMinIdle,
	}
	rdb, err := redisClient.New(context.Background(), redisCfg)
	if err != nil {
		zlog.Fatal("failed to init redis client", zap.Error(err))
	}
	defer func() {
		if cerr := rdb.Close(); cerr != nil {
			zlog.Warn("failed to close redis client", zap.Error(cerr))
		}
	}()
	if err := rdb.HealthCheck(context.Background()); err != nil {
		zlog.Fatal("redis health check failed", zap.Error(err))
	}
	zlog.Info("redis client initialized",
		zap.Int("pool_size", cfg.RedisPoolSize),
		zap.Int("min_idle_conns", cfg.RedisMinIdle),
	)

	storage, err := minioClient.New(
		context.Background(),
		cfg.MinIOURL,
		cfg.MinIORootUser,
		cfg.MinIORootPass,
		cfg.MinIOBucket,
		cfg.PresignTTLMin,
	)
	if err != nil {
		zlog.Fatal("failed to init minio client", zap.Error(err))
	}
	zlog.Info("minio client initialized",
		zap.String("bucket", cfg.MinIOBucket),
		zap.Int("presign_ttl_min", cfg.PresignTTLMin),
	)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestLogger(zlog))
	router.Use(middleware.RateLimiter(rdb, middleware.RateLimiterConfig{
		Limit:     cfg.RateLimitRequests,
		Window:    time.Duration(cfg.RateLimitWindowSec) * time.Second,
		KeyPrefix: "api:ratelimit",
	}))
	tokenIssuer := auth.NewTokenIssuer(cfg.JWTSecret, cfg.JWTAccessTTLMin, cfg.JWTRefreshTTLMin)
	refreshStore := auth.NewRefreshStore(rdb)
	registerHandler := auth.NewRegisterHandler(db)
	loginHandler := auth.NewLoginHandler(db, tokenIssuer, refreshStore)
	refreshHandler := auth.NewRefreshHandler(db, tokenIssuer, refreshStore)
	logoutHandler := auth.NewLogoutHandler(refreshStore)
	router.POST("/register", registerHandler.Handle)
	router.POST("/login", loginHandler.Handle)
	router.POST("/refresh", refreshHandler.Handle)
	router.POST("/logout", logoutHandler.Handle)
	router.GET("/health", func(c *gin.Context) {
		c.String(200, "OK")
	})

	protected := router.Group("/api")
	protected.Use(middleware.JWTAuth(cfg.JWTSecret))
	foldersHandler := folders.NewHandler(db)
	allowedMIMEs := strings.Split(cfg.UploadAllowedMIMEs, ",")
	filesHandler := files.NewHandler(db, storage, int64(cfg.UploadMaxSizeMB)*1024*1024, allowedMIMEs)
	protected.POST("/upload", filesHandler.Upload)
	protected.GET("/download", filesHandler.Download)
	protected.POST("/folders", foldersHandler.Create)
	protected.GET("/folders/resolve", foldersHandler.ResolvePath)
	protected.GET("/folders", foldersHandler.List)
	protected.GET("/folders/:id/breadcrumbs", foldersHandler.Breadcrumbs)
	protected.PATCH("/folders/:id", foldersHandler.Rename)
	protected.DELETE("/folders/:id", foldersHandler.Delete)
	protected.GET("/me", func(c *gin.Context) {
		userID, _ := c.Get(middleware.ContextUserIDKey)
		email, _ := c.Get(middleware.ContextUserEmailKey)
		c.JSON(200, gin.H{
			"user_id": userID,
			"email":   email,
		})
	})

	zlog.Info("cloudstore server starting", zap.String("port", cfg.Port))
	if err := router.Run(":" + cfg.Port); err != nil {
		zlog.Fatal("failed to start server", zap.Error(err))
	}
}
