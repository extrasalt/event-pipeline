# Mable Platform — Design Overview

Three independent Go modules, one vanilla JS tracking script. Architecture split lets each service build, deploy, and scale independently.

## Modules

| Module | Dir | What | Port |
|--------|-----|------|------|
| [Pipeline Library](pipeline/DESIGN.md) | `pipeline/` | Generic concurrent streaming pipeline. Zero dependencies. Stages: Map, Filter, Deduplicate, If, Generate, ChurnEnrich, Collect, Reduce, FanOut. | — |
| [Pipeline API](api/DESIGN.md) | `api/` | Event ingestion server. Gin HTTP routes, chDB embedded storage, async insert queue, Prometheus metrics, JWT auth, Grafana JSON endpoint. | `:8081` |
| [E-Commerce App](ecommerce/DESIGN.md) | `ecommerce/` | BFF (Go/Gin) serving a React 19 SPA at `/app`. Products API, JWT auth in HttpOnly cookies. Same-origin deploy. | `:8080` |
| [Tracking Script](script/DESIGN.md) | `script/` | Vanilla JS IIFE that emits 7 event types (page_view, click, add_to_cart, checkout, payment_info, purchase, lead). sendBeacon transport, zero framework deps. | — |

## Observability

- **Prometheus** scrapes `api:8081/metrics` for chDB insert rates, latencies, error rates, queue depth.
- **Grafana** reads Prometheus for operational panels and the `marcusolsson-json-datasource` plugin (from `api:8081/api/analytics/grafana`) for business metric panels.

See [Docker Compose](../docker/docker-compose.yml) for the full stack.
