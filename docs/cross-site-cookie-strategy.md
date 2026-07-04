# Cross-Site Cookie Strategy

## Current Configuration

Both the API and BFF set cookies with these parameters:

```
Set-Cookie: token=<jwt>; HttpOnly; Secure; SameSite=Lax; Path=/; Max-Age=86400
```

- **HttpOnly**: JavaScript cannot read the token. XSS cannot steal it.
- **Secure**: Only sent over HTTPS. Browsers allow `localhost` as a Secure context.
- **SameSite=Lax**: Cookie is sent on top-level navigations from other sites but not on subresource requests (images, iframes, fetch) initiated by other sites.
- **Path=/**: Cookie is sent on every request to the origin.
- **Max-Age=86400**: 24-hour expiry.

## Why SameSite=Lax

`SameSite=Lax` is chosen over `None` (which requires `Secure` and allows cross-site sending) and `Strict` (which blocks cookies on all cross-site requests including navigations).

| Setting | Cross-site GET (nav) | Cross-site POST (form) | Cross-site fetch/iframe |
|---------|---------------------|----------------------|------------------------|
| `None` | Sent | Sent | Sent |
| `Lax` | Sent | ⛔ Blocked | ⛔ Blocked |
| `Strict` | ⛔ Blocked | ⛔ Blocked | ⛔ Blocked |

**Why not `None`**: The tracking script sends events via `sendBeacon` from any origin. If the cookie were sent cross-site, a malicious site could make the browser include the user's session cookie when posting fake events. Since `sendBeacon` is a POST subresource request, `SameSite=Lax` blocks the cookie anyway — and that's correct: tracking events are anonymous and do not need authentication.

**Why not `Strict`**: Logging a user out of a third-party site should work without the user first clicking through. If a user navigates from the product site to the analytics dashboard via a link, `Strict` would not send the cookie on the initial GET — they'd appear logged out until they reload.

## Cookie Scope: Two Origins

The system has two separate origins in development:

| Service | Origin | Cookie scope |
|---------|--------|-------------|
| API (`api/`) | `localhost:8081` | `/track/events` (no auth), `/api/auth/*`, `/api/analytics` |
| BFF (`ecommerce/`) | `localhost:8080` | `/api/auth/*`, `/app/*` (SPA) |

## Phase model

### Phase 1 — Same-origin BFF (current)

BFF and SPA are deployed as a single unit. The BFF handles auth, serves the SPA, and proxies no requests to the API from the browser. The API is called only by the tracking script (cross-origin, no cookie) and by the Grafana/Prometheus stack (internal, no cookie).

### Phase 2 — Separate API domain

When the API is deployed as a standalone service (e.g., `api.mable.io`) separate from the BFF (`app.mable.io`):

- The API's `POST /api/auth/login` sets a cookie with `Domain=api.example.com; SameSite=None; Secure`
- The API's `GET /api/analytics` reads the cookie
- The tracking script sends events to `api.example.com` without cookies

This requires:

1. `SameSite=None; Secure` — allows the cookie to be sent cross-site from `app.example.com` to `api.example.com`
2. HTTPS in both places (required for `Secure` and `SameSite=None`)
3. CORS with `Access-Control-Allow-Credentials: true` on the API's auth and analytics endpoints
4. CSRF protection on `/api/auth/*` (e.g., `SameSite=Strict` on a separate CSRF token, or double-submit cookie pattern)

## Security considerations

| Threat | Mitigation |
|--------|-----------|
| XSS steals token | HttpOnly — cookie invisible to JS |
| CSRF on auth endpoints | SameSite=Lax blocks cross-site POST. State-changing endpoints (signup, login, logout) are POST-only. |
| CSRF on analytics | SameSite=Lax blocks cross-site GET from subresources. Analytics is read-only; information leakage is the only risk. |
| Token replay | 24-hour expiry, HMAC-signed server-side, no token storage in JS. |
| DNS rebinding | Not mitigated in current config. In production, validate `Host` header and use `SameSite=Strict` cookies with `__Host-` prefix. |
| Subdomain takeover | Not mitigated. In production, register the full domain and avoid wildcard DNS. |
