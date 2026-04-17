package main

import (
	"log"

	"cloudstore/backend/internal/config"
	_ "cloudstore/backend/internal/dbmigrate"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	log.Printf("migrations path: %s", cfg.MigrationsPath)

	router := gin.Default()
	router.GET("/health", func(c *gin.Context) {
		c.String(200, "OK")
	})

	log.Printf("cloudstore server starting on :%s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
