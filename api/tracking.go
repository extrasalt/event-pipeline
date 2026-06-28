package api

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/extrasalt/event-pipeline/pipeline"
)

// TrackingEvent represents a single browser tracking event sent by
// the client-side tracking script.
type TrackingEvent struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Timestamp time.Time      `json:"timestamp"`
	Data      map[string]any `json:"data"`
	UserAgent string         `json:"user_agent"`
	Timezone  string         `json:"timezone"`
	Location  string         `json:"location"`
	SessionID string         `json:"session_id"`
	ChurnProb float64        `json:"churn_prob,omitempty"`
}

// SetChurnScore implements pipeline.ChurnEnrichable.
func (e *TrackingEvent) SetChurnScore(s pipeline.ChurnScore) {
	e.ChurnProb = s.Probability
}

// trackingPipeline is the pipeline that processes all incoming
// tracking events. It validates, normalizes, filters to purchase
// events only, deduplicates by ID, and enriches with a churn score.
var trackingPipeline = pipeline.Pipeline[*TrackingEvent]{
	Stages: []pipeline.Stage[*TrackingEvent]{
		pipeline.Map("validate", func(e *TrackingEvent) (*TrackingEvent, error) {
			if e.Type == "" {
				return e, fmt.Errorf("empty event type")
			}
			return e, nil
		}),
		pipeline.Map("normalize", func(e *TrackingEvent) (*TrackingEvent, error) {
			if len(e.UserAgent) > 500 {
				e.UserAgent = e.UserAgent[:500]
			}
			return e, nil
		}),
		pipeline.Filter("purchaseOnly", func(e *TrackingEvent) (bool, error) {
			return e.Type == "purchase", nil
		}),
		pipeline.Deduplicate("dedup", func(e *TrackingEvent) string {
			return e.ID
		}, 10000),
		pipeline.ChurnEnrich[*TrackingEvent]("churnEnrich"),
	},
	Buffer: 4096,
}

// Store is an in-memory event store that also provides basic
// analytics snapshots. Thread-safe.
type Store struct {
	mu           sync.RWMutex
	events       []*TrackingEvent
	eventTypes   map[string]int64
	totalLatency time.Duration
	totalParams  int
	totalEvents  int64
}

// NewStore creates an empty Store.
func NewStore() *Store {
	return &Store{eventTypes: make(map[string]int64)}
}

// Add appends events to the store and updates analytics counters.
func (s *Store) Add(events []*TrackingEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for _, e := range events {
		s.events = append(s.events, e)
		s.eventTypes[e.Type]++
		s.totalLatency += now.Sub(e.Timestamp)
		s.totalParams += len(e.Data)
		s.totalEvents++
	}
}

// ProcessEvents feeds events through the tracking pipeline, stores
// the results, and returns the processed events along with per-stage
// metadata.
func (s *Store) ProcessEvents(ctx context.Context, events []*TrackingEvent) ([]*TrackingEvent, []pipeline.StageMeta, error) {
	in := make(chan *TrackingEvent, len(events))
	for _, e := range events {
		in <- e
	}
	close(in)

	out, collectMeta := trackingPipeline.Run(ctx, in)

	var processed []*TrackingEvent
	for e := range out {
		processed = append(processed, e)
	}

	s.Add(processed)
	return processed, collectMeta(), nil
}

// Analytics is a snapshot of the store for the GET /api/analytics
// endpoint.
type Analytics struct {
	TotalEvents      int64            `json:"total_events"`
	EventsByType     map[string]int64 `json:"events_by_type"`
	AvgCaptureTimeMs float64          `json:"avg_capture_time_ms"`
	AvgEventParams   float64          `json:"avg_event_params"`
}

// Snapshot returns a point-in-time Analytics view.
func (s *Store) Snapshot() Analytics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	avgCapture := 0.0
	avgParams := 0.0
	if s.totalEvents > 0 {
		avgCapture = float64(s.totalLatency.Milliseconds()) / float64(s.totalEvents)
		avgParams = float64(s.totalParams) / float64(s.totalEvents)
	}
	return Analytics{
		TotalEvents:      s.totalEvents,
		EventsByType:     s.eventTypes,
		AvgCaptureTimeMs: avgCapture,
		AvgEventParams:   avgParams,
	}
}
