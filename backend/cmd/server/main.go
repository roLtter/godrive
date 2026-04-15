package main

import (
	"log"

	"cloudstore/backend/internal/config"
	_ "cloudstore/backend/internal/dbmigrate"
	"github.com/gin-gonic/gin"
)

func main() {
	const port = "8080"

	log.Printf("migrations path: %s (set MIGRATIONS_PATH to override)", config.MigrationsPath())

	router := gin.Default()
	router.GET("/health", func(c *gin.Context) {
		c.String(200, "OK")
	})

	log.Printf("cloudstore server starting on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
