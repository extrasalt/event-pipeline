package main

import (
	"encoding/json"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed products.json
var productsData []byte
var products []map[string]any

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

	if err := json.Unmarshal(productsData, &products); err != nil || products == nil {
		products = []map[string]any{}
	}

	api := r.Group("/api")
	api.GET("/products", func(c *gin.Context) {
		c.JSON(http.StatusOK, products)
	})
	api.GET("/products/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
			return
		}
		target := float64(id)
		for _, p := range products {
			if p["id"] == target {
				c.JSON(http.StatusOK, p)
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
	})

	trackerAPIURL := os.Getenv("TRACKER_API_URL")
	if trackerAPIURL == "" {
		trackerAPIURL = "http://localhost:8081/track/events"
	}

	dist := filepath.Join("app", "dist")
	if info, err := os.Stat(dist); err == nil && info.IsDir() {
		indexPath := filepath.Join(dist, "index.html")
		indexBytes, err := os.ReadFile(indexPath)
		modifiedIndex := ""
		if err == nil {
			trackerScript := fmt.Sprintf(`<script>window._TRACKER_API=%q</script>`, trackerAPIURL)
			modifiedIndex = strings.Replace(string(indexBytes),
				`<script src="/app/script/tracker.js"></script>`,
				trackerScript+`<script src="/app/script/tracker.js"></script>`, 1)
		}

		httpFS := http.FileServer(http.Dir(dist))
		handler := func(c *gin.Context) {
			filePath := c.Param("filepath")
			if filePath == "" || filePath == "/" || filePath == "/index.html" {
				c.Header("Content-Type", "text/html; charset=utf-8")
				c.String(http.StatusOK, modifiedIndex)
				return
			}
			c.Request.URL.Path = filePath
			httpFS.ServeHTTP(c.Writer, c.Request)
		}

		r.GET("/app/*filepath", handler)
		r.HEAD("/app/*filepath", handler)
		r.GET("/app", func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, "/app/")
		})
		r.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, "/app/")
		})
		r.NoRoute(func(c *gin.Context) {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, modifiedIndex)
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
