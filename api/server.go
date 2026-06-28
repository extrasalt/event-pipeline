package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// NewServer creates a Gin engine with the following routes:
//
//	GET  /health         — {"status":"ok"}
//	POST /track/events   — ingest tracking events through the pipeline
//	GET  /api/analytics  — analytics snapshot
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
		c.JSON(http.StatusOK, store.Snapshot())
	})

	return r
}
