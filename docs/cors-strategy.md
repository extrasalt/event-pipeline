# CORS Strategy

## Architecture

The system has two HTTP servers:

| Service | Domain (dev) | Routes | Needs CORS |
|---------|-------------|--------|------------|
| API (`api/`) | `localhost:8081` | `/track/events`, `/api/auth/*`, `/api/analytics`, `/health`, `/metrics` | Yes — tracking script may embed on other origins |
| BFF (`ecommerce/`) | `localhost:8080` | `/app/*`, `/api/auth/*`, `/api/products` | No — SPA is same-origin with BFF |

## API CORS Policy

The API (Gin, `api/server.go`) applies a single CORS middleware to all routes:

```go
func corsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        origin := c.GetHeader("Origin")
        if origin != "" {
            c.Header("Access-Control-Allow-Origin", origin)
            c.Header("Access-Control-Allow-Credentials", "true")
        }
        c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(http.StatusNoContent)
            return
        }
        c.Next()
    }
}
```

### Rules

1. **Reflect origin** — `Access-Control-Allow-Origin` is set to the request's `Origin` header, not a hardcoded list. This is permissive but practical: the tracking script needs to work from any customer domain. In production this would be replaced with an allowlist.
2. **Credentials allowed** — `Access-Control-Allow-Credentials: true` enables cookie-based auth for analytics endpoints.
3. **Pre-flight** — OPTIONS requests return 204 with CORS headers and are aborted immediately.
4. **No origin → no CORS headers** — If no `Origin` header is present (e.g., internal requests), no CORS headers are set at all.

### Route-level distinctions

| Route | Credentials | Why |
|-------|-------------|-----|
| `POST /track/events` | Not needed | Tracking script sends no cookies; events are anonymous |
| `GET /api/analytics` | Required | JWT auth via HttpOnly cookie |
| `POST /api/auth/*` | Required | JWT cookie set/cleared |
| `GET /health`, `GET /metrics` | Not needed | No auth |

### BFF

The BFF does not set CORS headers. The SPA is served from the same origin (`localhost:8080`) via Go's static file server. All API calls from the SPA go to `/api/*` on the same origin — no cross-origin requests needed.

## Production considerations

| Concern | Recommendation |
|---------|---------------|
| Origin allowlist | Replace the reflect-origin with a configurable `ALLOWED_ORIGINS` env var (comma-separated list). Use `strings.Contains` or exact match. |
| Tracking ingress | `POST /track/events` does not need credentials. Consider a separate route group without `Access-Control-Allow-Credentials` to avoid preflight on simple requests. |
| `sendBeacon` preflights | `navigator.sendBeacon` with a JSON `Blob` triggers a preflight. In production, support `text/plain` content type to make it a simple request and skip preflight entirely. |
