package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewServer(store *Store) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.POST("/track/events", func(c *gin.Context) {
		var events []*TrackingEvent
		if err := c.ShouldBindJSON(&events); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
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

	r.GET("/api/analytics", func(c *gin.Context) {
		snap, err := store.Snapshot(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, snap)
	})

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	return r
}
