package api

import (
	"context"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
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

func NewServer(store *Store) *gin.Engine {
	r := gin.Default()
	r.Use(corsMiddleware())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.POST("/track/events", func(c *gin.Context) {
		var events []*TrackingEvent
		if err := c.ShouldBindJSON(&events); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		clientIP := c.ClientIP()
		requestOrigin := c.GetHeader("Origin")
		for _, e := range events {
			e.IP = clientIP
			if e.Source == "" {
				e.Source = "unknown"
			}
			if requestOrigin != "" {
				e.Origin = requestOrigin
			}
		}

		processed, meta, err := store.ProcessEvents(context.Background(), events)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"received":  len(events),
			"processed": len(processed),
			"dropped":   len(events) - len(processed),
			"metadata":  meta,
		})
	})

	auth := r.Group("/api/auth")
	{
		auth.POST("/signup", handleSignup)
		auth.POST("/login", handleLogin)
		auth.POST("/logout", handleLogout)
		auth.GET("/me", authMiddleware(), handleMe)
	}

	r.GET("/api/analytics", authMiddleware(), func(c *gin.Context) {
		source := c.DefaultQuery("source", "")
		snap, err := store.Snapshot(c.Request.Context(), source)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, snap)
	})

	r.GET("/api/analytics/grafana", func(c *gin.Context) {
		metric := c.DefaultQuery("metric", "")
		source := c.DefaultQuery("source", "")

		snap, err := store.Snapshot(c.Request.Context(), source)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		switch metric {
		case "total_events":
			c.JSON(http.StatusOK, []any{map[string]any{"value": snap.TotalEvents}})
		case "avg_capture_time_ms":
			c.JSON(http.StatusOK, []any{map[string]any{"value": snap.AvgCaptureTimeMs}})
		case "avg_event_params":
			c.JSON(http.StatusOK, []any{map[string]any{"value": snap.AvgEventParams}})
		case "events_by_source":
			var result []map[string]any
			for src, count := range snap.EventsBySource {
				result = append(result, map[string]any{"source": src, "count": count})
			}
			sort.Slice(result, func(i, j int) bool {
				return result[i]["count"].(uint64) > result[j]["count"].(uint64)
			})
			c.JSON(http.StatusOK, result)
		case "events_by_type":
			var result []map[string]any
			for typ, count := range snap.EventsByType {
				result = append(result, map[string]any{"type": typ, "count": count})
			}
			sort.Slice(result, func(i, j int) bool {
				return result[i]["count"].(uint64) > result[j]["count"].(uint64)
			})
			c.JSON(http.StatusOK, result)
		case "events_over_time":
			var result []map[string]any
			for _, tb := range snap.EventsOverTime {
				t, err := time.Parse("2006-01-02", tb.Date)
				if err != nil {
					continue
				}
				result = append(result, map[string]any{"time": t.UnixMilli(), "value": tb.Count})
			}
			c.JSON(http.StatusOK, result)
		default:
			c.JSON(http.StatusOK, []any{})
		}
	})

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	return r
}
