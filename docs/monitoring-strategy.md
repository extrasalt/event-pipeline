# Monitoring Strategy

## Layers

```
┌──────────────────────────────────────────────────────────────┐
│                        Grafana                               │
│  ┌─────────────────┐  ┌──────────────┐  ┌────────────────┐  │
│  │ Event rate       │  │ Insert       │  │ Queue depth    │  │
│  │ (events/s)       │  │ latency p50  │  │ (backlog)      │  │
│  │                  │  │ /p90/p99     │  │                │  │
│  └─────────────────┘  └──────────────┘  └────────────────┘  │
└───────────────────────────┬──────────────────────────────────┘
                            │ Prometheus (scrape :8081/metrics)
┌───────────────────────────┴──────────────────────────────────┐
│                       API (Go)                                │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────┐   │
│  │ Prometheus  │  │ Structured   │  │ Error logging      │   │
│  │ counters    │  │ JSON logs    │  │ (stderr)           │   │
│  └─────────────┘  └──────────────┘  └────────────────────┘   │
└───────────────────────────────────────────────────────────────┘
┌──────────────────────────────────────────────────────────────┐
│                     SPA (Browser)                             │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────┐   │
│  │ Error       │  │ Structured   │  │ Performance        │   │
│  │ Boundary    │  │ JSON logger  │  │ (console)          │   │
│  └─────────────┘  └──────────────┘  └────────────────────┘   │
└───────────────────────────────────────────────────────────────┘
```

## 1. Backend (API)

### Prometheus Metrics

Exposed at `GET /metrics` on port `:8081`. Instrumented in `api/store.go`.

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `chdb_inserts_total` | Counter | — | Successful chDB batch inserts |
| `chdb_insert_errors_total` | Counter | — | Failed inserts after all retries |
| `chdb_insert_latency_seconds` | Histogram | — | Latency of batch inserts (def buckets) |
| `chdb_queue_depth` | Gauge | — | Events waiting in the insert queue |

### Grafana Dashboard

Auto-provisioned at `docker/grafana/dashboards/pipeline-dashboard.json`. Three panels:

1. **Insert Rate** — `rate(chdb_inserts_total[1m])`
2. **Insert Latency** — `histogram_quantile(0.50/0.90/0.99, rate(chdb_insert_latency_seconds_bucket[1m]))`
3. **Error Rate** — `rate(chdb_insert_errors_total[1m])`
4. **Queue Depth** — `chdb_queue_depth`

The dashboard uses the JSON API datasource (pointing at `api:8081/api/analytics/grafana`) for event-type breakdowns, not just Prometheus metrics.

### Health Checks

`GET /health` returns `{"status":"ok"}`. No dependency checks (chDB is embedded, always available if the process is running).

### Structured Logging

The Go API uses Gin's default logger (stdout) plus explicit `fmt.Printf` for errors in `store.go` (flush errors, queue full drops). In production, replace with a structured logger (zap/slog) writing JSON to stdout.

## 2. Frontend (SPA)

### Error Boundary

`ecommerce/app/src/components/ErrorBoundary.jsx` wraps all routes. On a render crash:

1. Calls `logger.error("React render error", { error, componentStack })`
2. Renders a fallback UI with "Something went wrong" message and a retry button
3. The error is logged to the browser console as structured JSON

### Structured Logger

`ecommerce/app/src/lib/logger.js` — four levels (debug, info, warn, error). Every entry is a JSON object with `timestamp`, `level`, `message`, `meta`. Output via `console.log`/`console.error`.

Usage across the app:
- **Store actions**: `logger.info("login success", { email })` in authStore
- **API calls**: `logger.error("fetch failed", { url, status })` in api.js
- **Navigation**: `logger.debug("page view", { path })` in App.jsx

### Performance Monitoring

Current: manual console instrumentation. No real-user monitoring (RUM) agent attached. Options for production:

| Tool | Pros | Cons |
|------|------|------|
| Web Vitals (library) | Free, measures LCP/CLS/INP | Raw data, no dashboard |
| Sentry RUM | Error + performance, one agent | Paid after free tier |
| Grafana Faro | Open source, integrates with existing Grafana | Self-hosted |
| Datadog RUM | Full-featured | Expensive |

### Client-Side Error Tracking

Current: `window.onerror` and `window.onunhandledrejection` are not explicitly registered. The ErrorBoundary catches React render errors but not async errors outside React (e.g., `setTimeout`, event handlers). In production, add a global handler that pipes to the structured logger.

## 3. Alerts (to configure)

| Condition | Severity | Action |
|-----------|----------|--------|
| `chdb_insert_errors_total` rate > 0 | Critical | Check chDB disk / schema / connectivity |
| `chdb_queue_depth` > 50K | Warning | Events backing up; scale up consumer |
| `chdb_queue_depth` > 100K | Critical | Consumer likely stuck; restart |
| `rate(chdb_inserts_total[5m])` == 0 | Warning | No events arriving; check tracking script / network |

## Gaps

- [ ] No global `window.onerror` / `onunhandledrejection` handler in the SPA
- [ ] No RUM / Web Vitals tracking
- [ ] No structured logging in the Go API (uses `fmt.Printf` for errors)
- [ ] No alerting rules configured in Grafana (ad-hoc only)
- [ ] No uptime / health check monitoring external to the stack
