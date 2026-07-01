package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/extrasalt/event-pipeline/pipeline"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Config struct {
	Host          string
	Port          string
	Database      string
	Table         string
	BatchSize     int
	FlushInterval time.Duration
	MaxRetries    int
	QueueSize     int
}

func ConfigFromEnv() Config {
	getEnv := func(key, fallback string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return fallback
	}
	getInt := func(key string, fallback int) int {
		if v := os.Getenv(key); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				return n
			}
		}
		return fallback
	}
	getDuration := func(key string, fallback time.Duration) time.Duration {
		if v := os.Getenv(key); v != "" {
			if d, err := time.ParseDuration(v); err == nil {
				return d
			}
		}
		return fallback
	}
	return Config{
		Host:          getEnv("CLICKHOUSE_HOST", "localhost"),
		Port:          getEnv("CLICKHOUSE_PORT", "9000"),
		Database:      getEnv("CLICKHOUSE_DB", "default"),
		Table:         getEnv("CLICKHOUSE_TABLE", "events"),
		BatchSize:     getInt("CLICKHOUSE_BATCH_SIZE", 10000),
		FlushInterval: getDuration("CLICKHOUSE_FLUSH_INTERVAL", 1*time.Second),
		MaxRetries:    getInt("CLICKHOUSE_MAX_RETRIES", 3),
		QueueSize:     getInt("CLICKHOUSE_QUEUE_SIZE", 100000),
	}
}

type Store struct {
	conn   clickhouse.Conn
	table  string
	events chan *TrackingEvent
	done   chan struct{}

	batchSize     int
	flushInterval time.Duration
	maxRetries    int

	inserts    prometheus.Counter
	insertErrs prometheus.Counter
	insertLat  prometheus.Histogram
	queueDepth prometheus.Gauge
}

func NewStore(ctx context.Context, cfg Config) (*Store, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{cfg.Host + ":" + cfg.Port},
		Auth: clickhouse.Auth{Database: cfg.Database},
		Settings: map[string]any{
			"async_insert":              1,
			"wait_for_async_insert":     0,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("clickhouse connect: %w", err)
	}
	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("clickhouse ping: %w", err)
	}
	if err := conn.Exec(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id          String,
			type        String,
			timestamp   DateTime64(3),
			data        String,
			user_agent  String,
			timezone    String,
			location    String,
			session_id  String,
			churn_prob  Float64,
			param_count UInt32,
			inserted_at DateTime DEFAULT now()
		) ENGINE = MergeTree()
		ORDER BY (type, timestamp)
	`, cfg.Table)); err != nil {
		return nil, fmt.Errorf("clickhouse migrate: %w", err)
	}

	s := &Store{
		conn:          conn,
		table:         cfg.Table,
		events:        make(chan *TrackingEvent, cfg.QueueSize),
		done:          make(chan struct{}),
		batchSize:     cfg.BatchSize,
		flushInterval: cfg.FlushInterval,
		maxRetries:    cfg.MaxRetries,
		inserts: promauto.NewCounter(prometheus.CounterOpts{
			Name: "clickhouse_inserts_total",
			Help: "Total successful ClickHouse insert batches.",
		}),
		insertErrs: promauto.NewCounter(prometheus.CounterOpts{
			Name: "clickhouse_insert_errors_total",
			Help: "Total ClickHouse insert batch errors after retries.",
		}),
		insertLat: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "clickhouse_insert_latency_seconds",
			Help:    "Latency of ClickHouse batch inserts.",
			Buckets: prometheus.DefBuckets,
		}),
		queueDepth: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "clickhouse_queue_depth",
			Help: "Current number of events waiting in the insert queue.",
		}),
	}

	go s.run(ctx)

	return s, nil
}

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

	for _, e := range processed {
		select {
		case s.events <- e:
		default:
			s.insertErrs.Inc()
		}
	}
	s.queueDepth.Set(float64(len(s.events)))

	return processed, collectMeta(), nil
}

func (s *Store) run(ctx context.Context) {
	buf := make([]*TrackingEvent, 0, s.batchSize)
	ticker := time.NewTicker(s.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.flush(buf)
			close(s.done)
			return
		case e := <-s.events:
			buf = append(buf, e)
			if len(buf) >= s.batchSize {
				s.flush(buf)
				buf = make([]*TrackingEvent, 0, s.batchSize)
			}
		case <-ticker.C:
			if len(buf) > 0 {
				s.flush(buf)
				buf = make([]*TrackingEvent, 0, s.batchSize)
			}
		}
	}
}

func (s *Store) flush(events []*TrackingEvent) {
	start := time.Now()
	err := s.insertWithRetry(context.Background(), events)
	if err != nil {
		s.insertErrs.Inc()
		return
	}
	s.inserts.Inc()
	s.insertLat.Observe(time.Since(start).Seconds())
}

func (s *Store) insertWithRetry(ctx context.Context, events []*TrackingEvent) error {
	var lastErr error
	for attempt := range s.maxRetries {
		if attempt > 0 {
			time.Sleep(backoff(attempt))
		}
		err := s.insertBatch(ctx, events)
		if err == nil {
			return nil
		}
		lastErr = err
	}
	return lastErr
}

func backoff(attempt int) time.Duration {
	jitter := time.Duration(rand.Int63n(100)) * time.Millisecond
	return time.Duration(math.Pow(2, float64(attempt)))*100*time.Millisecond + jitter
}

func (s *Store) insertBatch(ctx context.Context, events []*TrackingEvent) error {
	batch, err := s.conn.PrepareBatch(ctx, "INSERT INTO "+s.table)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	for _, e := range events {
		dataJSON, _ := json.Marshal(e.Data)
		if err := batch.Append(
			e.ID,
			e.Type,
			e.Timestamp,
			string(dataJSON),
			e.UserAgent,
			e.Timezone,
			e.Location,
			e.SessionID,
			e.ChurnProb,
			uint32(len(e.Data)),
		); err != nil {
			return fmt.Errorf("append: %w", err)
		}
	}
	return batch.Send()
}

func (s *Store) Close() {
	<-s.done
	s.conn.Close()
}

type Analytics struct {
	TotalEvents      uint64            `json:"total_events"`
	EventsByType     map[string]uint64 `json:"events_by_type"`
	AvgCaptureTimeMs float64           `json:"avg_capture_time_ms"`
	AvgEventParams   float64           `json:"avg_event_params"`
}

func (s *Store) Snapshot(ctx context.Context) (Analytics, error) {
	var a Analytics

	row := s.conn.QueryRow(ctx, "SELECT count() FROM "+s.table)
	if err := row.Scan(&a.TotalEvents); err != nil {
		return a, fmt.Errorf("count: %w", err)
	}

	rows, err := s.conn.Query(ctx, "SELECT type, count() FROM "+s.table+" GROUP BY type")
	if err != nil {
		return a, fmt.Errorf("group by type: %w", err)
	}
	defer rows.Close()
	a.EventsByType = make(map[string]uint64)
	for rows.Next() {
		var typ string
		var cnt uint64
		if err := rows.Scan(&typ, &cnt); err != nil {
			return a, fmt.Errorf("scan type row: %w", err)
		}
		a.EventsByType[typ] = cnt
	}

	row = s.conn.QueryRow(ctx, "SELECT avg(dateDiff('millisecond', timestamp, inserted_at)) FROM "+s.table)
	var avgCapture *float64
	if err := row.Scan(&avgCapture); err != nil {
		return a, fmt.Errorf("avg capture: %w", err)
	}
	if avgCapture != nil {
		a.AvgCaptureTimeMs = *avgCapture
	}

	row = s.conn.QueryRow(ctx, "SELECT avg(param_count) FROM "+s.table)
	var avgParams *float64
	if err := row.Scan(&avgParams); err != nil {
		return a, fmt.Errorf("avg params: %w", err)
	}
	if avgParams != nil {
		a.AvgEventParams = *avgParams
	}

	return a, nil
}
