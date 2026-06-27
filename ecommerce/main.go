package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	dist := filepath.Join("app", "dist")
	if info, err := os.Stat(dist); err == nil && info.IsDir() {
		r.Static("/app", dist)
		r.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, "/app/")
		})
		r.NoRoute(func(c *gin.Context) {
			c.File(filepath.Join(dist, "index.html"))
		})
	}

	log.Println("starting BFF server on :8080")
	if err := r.Run(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
