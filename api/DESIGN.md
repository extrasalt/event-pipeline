# Pipeline API — Design Document

## Overview

Gin-based HTTP server that ingests browser tracking events, runs them through the pipeline library (validate → normalize → deduplicate → churn enrich), stores them in chDB (embedded ClickHouse via purego/dlopen), and exposes analytics endpoints. Prometheus instrumentation built in.

## Architecture

```
POST /track/events ──► trackingPipeline ──► async insert queue ──► chDB
                        │                       │                   │
                      Meta()                  chdb_inserts_total    │
                      returned                chdb_insert_errors    │
                      in response             chdb_insert_latency   │
                                              chdb_queue_depth      │
                                                                     ▼
GET /api/analytics ──► Snapshot() ◄── SQL queries ─────────────────┘
GET /api/analytics/grafana?metric= ──► Snapshot() ──► records JSON
```

## Routes

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/track/events` | No | Ingest batch of tracking events |
| GET | `/api/analytics` | Yes | Full analytics snapshot as JSON |
| GET | `/api/analytics/grafana?metric=` | No | Per-metric records for Grafana JSON API datasource |
| POST | `/api/auth/signup` | No | Create user, set JWT cookie |
| POST | `/api/auth/login` | No | Authenticate, set JWT cookie |
| POST | `/api/auth/logout` | No | Clear JWT cookie |
| GET | `/api/auth/me` | Yes | Return user from JWT claims |
| GET | `/metrics` | No | Prometheus scrape endpoint |
| GET | `/health` | No | Health check |

## Pipeline Stages

1. **validate** (`Map`) — rejects events with empty type; error = dropped
2. **normalize** (`Map`) — truncates UserAgent (500) and IP (45) to max lengths
3. **dedup** (`Deduplicate`) — in-memory seen-set keyed by event ID, capacity 10,000
4. **churnEnrich** (`ChurnEnrich`) — assigns random churn probability to every event

All event types pass through (no purchase-only filter).

## Storage (chDB)

**chDB** is an embedded ClickHouse loaded via `purego` dlopen — no external database process. Data stored on disk at `CHDB_DATA_PATH` (default `/tmp/chdb`).

### Schema

| Column | Type | Notes |
|--------|------|-------|
| id | String | UUID from tracking script |
| type | String | event type (purchase, page_view, click, etc.) |
| timestamp | DateTime64(3) | client-side event time |
| data | String | JSON-encoded arbitrary key-value pairs |
| user_agent | String | truncated to 500 chars |
| ip | String | server-side ClientIP, truncated to 45 |
| timezone | String | from client |
| location | String | from client |
| session_id | String | from tracking script session |
| churn_prob | Float64 | simulated churn enrichment |
| param_count | UInt32 | len(data) |
| inserted_at | DateTime | server-side insert time, DEFAULT now() |

Engine: `MergeTree ORDER BY (type, timestamp)`

### Insert Flow

Events flow through the pipeline synchronously per `/track/events` request, then are enqueued to a buffered channel. A background goroutine batches them (max `CHDB_BATCH_SIZE`, default 100) and flushes on batch full or `CHDB_FLUSH_INTERVAL` (default 1s). Retry with exponential backoff + jitter on failure (max `CHDB_MAX_RETRIES`, default 3).

### Metrics (Prometheus)

| Metric | Type | Labels |
|--------|------|--------|
| `chdb_inserts_total` | Counter | — |
| `chdb_insert_errors_total` | Counter | — |
| `chdb_insert_latency_seconds` | Histogram | default buckets |
| `chdb_queue_depth` | Gauge | — |

## Analytics

`Snapshot()` runs 5 SQL queries against chDB:

1. `SELECT count() FROM events` — total events
2. `SELECT type, count() FROM events GROUP BY type` — events by type
3. `SELECT date, count() FROM events GROUP BY date ORDER BY date` — events over time
4. `SELECT avg(dateDiff('millisecond', timestamp, inserted_at))` — avg capture-to-insert latency
5. `SELECT avg(param_count)` — avg event parameter count

### Grafana Endpoint

`/api/analytics/grafana?metric=<name>` returns records-format JSON consumed by the `marcusolsson-json-datasource` plugin (v1.4.0). Response is always an array of objects:

| metric | Response shape |
|--------|---------------|
| `total_events` | `[{"value": N}]` |
| `avg_capture_time_ms` | `[{"value": N}]` |
| `avg_event_params` | `[{"value": N}]` |
| `events_by_type` | `[{"type": "purchase", "count": N}, ...]` |
| `events_over_time` | `[{"time": epoch_ms, "value": N}, ...]` |

No auth — designed for Grafana server-side datasource provisioning.

## Auth

Custom JWT (~40 lines stdlib, no library). HMAC-SHA256, 24-hour expiry. Token in HttpOnly cookie (`token`), SameSite=Lax, Secure. Separate JWT secret from the BFF (`api-demo-secret-change-in-production`). In-memory user store with `sync.RWMutex`. Passwords in plaintext — demo scope only.

## Docker

Multi-stage Dockerfile: `golang:1.25` builder downloads `libchdb.so` for the target arch, runtime is `debian:bookworm-slim`. The shared library is loaded via purego dlopen — the Go binary is statically linked (`CGO_ENABLED=0`).

## Dependencies

- `github.com/gin-gonic/gin` — HTTP framework
- `github.com/chdb-io/chdb-go` — chDB driver (database/sql interface)
- `github.com/prometheus/client_golang` — Prometheus metrics
- `github.com/extrasalt/event-pipeline/pipeline` — local pipeline library (replace directive)
- Transitives include `ebitengine/purego` (dlopen), `parquet-go`, ClickHouse protocol libs
