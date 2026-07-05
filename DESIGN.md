# Design Document

## Architecture
Browser SPA + Go BFF (same-origin deployment, single binary) for the storefront. 
A separate Go API ingests tracking events through a streaming pipeline library and stores them in chDB.

```text
Browser → BFF (Render :8080)       → SPA, auth, products API
        → Tracking API (Railway)    → pipeline → chDB
```

**Repository structure** (per spec): `ecommerce/` (SPA + BFF), `api/` (tracking API), `pipeline/` (reusable streaming library), `script/` (tracker JS), `links.md` (deployment URLs).

## Key Decisions & Trade-offs
1. The demo ecommerce app has a Go backend that serves the products from a JSON over generic fakestoreapi. The data is from fakestoreapi.com. We embed the json during the build to avoid http call failure on init failing the deployment. 
2. using chdb with persistent volume backing because deploying full clickhouse on our chosen hosting provider is problematic. This comes with performance penalties and the failure modes mentioned below. Considered the clickhouse cloud free trial but that seemed like an overkill. This limits API horizontal scalability but since chdb and clickhouse are interchangeable, chdb can be swapped out if scale ever becomes a problem.
3. Using vite+ over vite. 

## Failure Modes

- **chDB volume full** → inserts fail; queue overflow drops events silently
- **Pipeline buffer full** → events dropped; `dropped` counter incremented
- **API restart** → in-memory event queue lost; Prometheus counters reset; chDB data persists (volume mount)
- **Dedup capacity reset** → duplicate events pass through after seen-map clears
- **JWT hardcoded secret** → public repo; anyone can forge tokens. Must be moved to env var
- **sendBeacon failure** → fallback to `fetch` with `keepalive` + `.catch()` — never blocks UI

## Scaling Strategy

- **Pipeline**: configurable buffer depth, batch size, worker fan-out (`FanOut` stage). Benchmarked at 1M events.
- **chDB**: single-node embedded — scale requires replacing with ClickHouse cluster. Schema ordered on `(type, timestamp)`.
- **API**: stateless except chDB; horizontal scale with NFS volume or consistent hashing of source ID.
- **BFF**: stateless (in-memory auth); scale horizontally with shared JWT secret.

## Team Split (3 Engineers)

| Engineer | Focus |
|----------|-------|
| **Frontend** | SPA (React + Zustand + Tailwind), Playwright E2E, tracking JS, accessibility |
| **Platform** | Go API + pipeline library, chDB, benchmarks, Prometheus, Grafana |
| **DevOps** | Docker Compose, Render + Railway deployment, env vars, volumes, CI/CD |

Risk: Bus factor

## First Two-Week Execution Plan

1. **Week 1**: Pipeline library (Map, Filter, Deduplicate, Reduce, If, Generate, Collect) + tests + benchmarks. Go API skeleton with `/track/events` wired to pipeline + chDB. Docker Compose with Prometheus + Grafana.
2. **Week 2**: SPA (all pages, auth, cart, checkout), BFF (auth, static serving, product API), tracking JS (7 event types, Beacon API). E2E tests. Deploy to Render + Railway. Grafana dashboard. Design doc + review doc.
