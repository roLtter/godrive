package main

import (
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	const port = "8080"

	router := gin.Default()
	router.GET("/health", func(c *gin.Context) {
		c.String(200, "OK")
	})

	log.Printf("cloudstore server starting on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
