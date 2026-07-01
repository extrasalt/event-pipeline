# event-pipeline

## Quick start

```sh
docker compose -f docker/docker-compose.yml up --build
```

Starts ClickHouse, pipeline API (:8081), e-commerce app (:8080), Prometheus (:9090), Grafana (:3000).

## Manual (development)

### Prerequisites

- Go 1.25+
- `vp` (Vite+) — `npm install -g vite-plus` or your package manager of choice

### Dependencies

```sh
docker compose -f docker/docker-compose.yml up clickhouse
```

The API requires ClickHouse on :9000. Override via `CLICKHOUSE_HOST`, `CLICKHOUSE_PORT`, `CLICKHOUSE_DB`, `CLICKHOUSE_TABLE`.

### 1. Frontend

```sh
cd ecommerce/app
vp install
vp build
```

### 2. BFF (e-commerce + auth)

```sh
go run ./ecommerce
```

Listens on :8080. Serves the frontend at `/app`, handles signup/login/logout with JWT in HttpOnly cookies.

### 3. Pipeline API

```sh
go run ./api/cmd/server
```

Listens on :8081. Ingests tracking events through the pipeline library, persists to ClickHouse, serves analytics.

## Tests

```sh
cd pipeline && go test -race -bench=. -benchtime=10ms
cd ecommerce/app && vp run test:e2e
```

## Project layout

```
ecommerce/     BFF + React SPA + Playwright tests
script/        Tracking script (vanilla JS, Beacon API)
api/           Go API — event ingestion, pipeline processing, analytics
pipeline/      Generic concurrent pipeline library (zero external deps)
docker/        Docker Compose stack (ClickHouse, Prometheus, Grafana)
docs/          Design docs, benchmarks, auth strategy
links/         Deployment URLs (gitignored)
```
