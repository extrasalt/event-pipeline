package main

import (
	_ "embed"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
)

//go:embed products.json
var productsJSON []byte

func main() {
	r := gin.Default()

	r.Use(corsMiddleware())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	auth := r.Group("/api/auth")
	auth.POST("/signup", handleSignup)
	auth.POST("/login", handleLogin)
	auth.POST("/logout", handleLogout)
	auth.GET("/me", authMiddleware, handleMe)

	api := r.Group("/api")
	api.GET("/products", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json", productsJSON)
	})
	api.GET("/products/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
			return
		}
		var products []map[string]any
		if err := json.Unmarshal(productsJSON, &products); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse products"})
			return
		}
		for _, p := range products {
			if p["id"] == id {
				c.JSON(http.StatusOK, p)
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
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

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "http://localhost:5173" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
