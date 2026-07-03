package api

import (
	"fmt"
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
	IP        string         `json:"-"`
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
			if len(e.IP) > 45 {
				e.IP = e.IP[:45]
			}
			return e, nil
		}),
		pipeline.Deduplicate("dedup", func(e *TrackingEvent) string {
			return e.ID
		}, 10000),
		pipeline.ChurnEnrich[*TrackingEvent]("churnEnrich"),
	},
	Buffer: 4096,
}
