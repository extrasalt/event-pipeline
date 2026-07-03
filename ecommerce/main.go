package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
)

var (
	productsCache []map[string]any
	productsOnce  sync.Once
)

func loadProducts() ([]map[string]any, error) {
	var err error
	productsOnce.Do(func() {
		resp, e := http.Get("https://fakestoreapi.com/products")
		if e != nil {
			err = e
			return
		}
		defer resp.Body.Close()
		var products []map[string]any
		if e := json.NewDecoder(resp.Body).Decode(&products); e != nil {
			err = e
			return
		}
		productsCache = products
	})
	return productsCache, err
}

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
		products, err := loadProducts()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load products"})
			return
		}
		c.JSON(http.StatusOK, products)
	})
	api.GET("/products/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
			return
		}
		products, err := loadProducts()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load products"})
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
