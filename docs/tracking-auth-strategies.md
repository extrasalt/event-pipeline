# Tracking API Authentication Strategy

## Context

The system has two distinct consumers:
- **Tracking script** — embedded on customer sites, sends events via `navigator.sendBeacon` (no custom headers), request bodies are JSON
- **Analytics dashboard** — customers logging in via browser to view event data

The spec says "Authentication: Signup, Login, JWT in HttpOnly cookies" under Go API. The question is: which consumers need auth, and how?

---

## Strategy A: Public Tracking ID (Google Analytics model)

### How it works
- Each customer gets a public tracking ID embedded in the script config: `window._TRACKER_ID = "MAB-XXXXX"`
- Sent with every event as a field in the JSON body
- The API uses it to route events to the right customer tenant
- No authentication on `POST /track/events` — the ID is a routing key, not a credential
- JWT in HttpOnly cookies on `GET /api/analytics` for dashboard access

### Pros
- Works with `sendBeacon` without modification (no custom headers)
- Simple for customers to set up — copy-paste a snippet
- Scales to many tenants
- Dashboard auth is separate and properly secured

### Cons
- Anyone can submit events with a valid tracking ID (it's visible in page source)
- No way to prevent spam or data injection from third parties
- Relies on server-side validation (rate limits, schema checks, anomaly detection)
- Attribution can be spoofed

### Real-world comparison
- **Google Analytics**: Uses GA4 measurement IDs (`G-XXXXXXXX`) or UA property IDs (`UA-XXXXX-Y`). Public in the snippet, no auth on the collection endpoint (`/g/collect`). Anti-abuse is purely server-side (rate limits, bot detection).
- **Mixpanel**: Uses a public token in the snippet. Same model — visible in page source, no auth on ingestion.
- **Amplitude**: Uses an API key in the snippet. Also public, no auth on ingestion from browsers.

**Verdict**: Industry standard for client-side analytics ingestion. The tracking ID is a namespace, not a firewall.

---

## Strategy B: Ingress Token in Request Body

### How it works
- Each customer gets a write-only API token (random 32+ character string)
- The token is embedded in the script config: `window._TRACKER_KEY = "mab_xxxxxxxxxxxxx"`
- Sent with every event as a field in the JSON body
- API validates the token before accepting the event
- JWT on analytics endpoints

### Pros
- Token is a credential — harder for attackers to guess than a short public ID
- Can rotate tokens per customer without changing the tracking ID
- Non-repudiation at the tenant level: if you see a valid token, it came from someone who had access to the snippet config
- Still works with `sendBeacon` (token is in body, not headers)

### Cons
- Token is visible in page source — same leak surface as a tracking ID
- Customers who don't want to expose a secret must add server-side proxying (their backend forwards to the API with a secret key)
- Adds token validation to the hot path of every event
- Token management (generation, rotation, revocation) adds operational overhead

### Real-world comparison
- **Stripe**: Public publishable key in the client (`pk_test_...`). Backed by a secret key (`sk_test_...`) that never leaves the server. Ingress is the public key — anyone with it can create tokens. The actual auth happens server-side where Stripe validates the secret key.
- **Segment**: Uses a public write key in the client. Same model — visible in page source, backends authenticate with secret keys.
- **LogRocket**: Public API ID in the snippet. No auth on ingestion.

**Verdict**: This is Google Analytics with a slightly longer key. In practice, a public token visible in the page source is not meaningfully more secure than a public tracking ID against a determined attacker.

---

## Strategy C: Shared JWT Secret (BFF signs, API verifies)

### How it works
- BFF and API share a JWT secret
- User visits the e-commerce app, logs in via BFF, gets a JWT in an HttpOnly cookie for the BFF domain
- Tracking script reads the JWT from a non-cookie source (e.g., a `<meta>` tag rendered by the BFF, or a JS variable set by the BFF)
- JWT is sent with every event in the JSON body
- API validates the JWT on every `/track/events` request

### Pros
- JWT means the request is tied to an authenticated user session
- The spec requirement for JWT is satisfied on the tracking endpoint
- No new token types to manage — reuses the existing auth

### Cons
- **Only works inside the demo app.** A third-party customer cannot use this — their own BFF would need to issue compatible JWTs signed by the shared secret
- Sharing one JWT secret across customer deployments is a terrible security practice
- Every customer would need to run their own BFF to sign JWTs — defeats the goal of a self-service tracking snippet
- `sendBeacon` can't inspect response — if the JWT expires mid-session, events silently fail
- The JWT is readable by JavaScript (it's in a meta tag or JS variable, not an HttpOnly cookie) — contradicts the spec's "never store in on the client" principle

**Verdict**: Feasible but only for a single-tenant demo. Breaks completely for multi-tenant tracking as a product.

---

## Strategy D: Two-Tier (Public Tracking ID + Secret Backend Key)

### How it works
- Each customer has a **public tracking ID** (visible, in the snippet) and a **secret backend key** (never in the browser)
- The tracking script sends events with just the tracking ID to the API
- The API accepts them unconditionally (like Strategy A) but buffers them as "unverified"
- Customers optionally set up a backend webhook or poll endpoint to verify their event stream
- Analytics dashboard uses JWT (like all strategies)

### Pros
- Simple snippet for customers who don't need strong guarantees
- Power users can set up verification for fraud detection
- No latency added to the hot path

### Cons
- No real auth on ingestion — the backend key is a monitoring/debugging tool, not a security mechanism
- Complex for a demo
- No major product does this

**Verdict**: Not worth the complexity.

---

## Recommended: Strategy A + Server-Side Guardrails

Use the public tracking ID model (Strategy A) because:

1. **Industry standard** — every major analytics product (GA, Mixpanel, Amplitude, Heap) works this way. There is no meaningful authentication on browser-side event ingestion.
2. **Works with `sendBeacon`** — no headers, no special transport. Just a field in the JSON.
3. **Tenant isolation** — the tracking ID scopes events to the right customer without precluding future auth improvements.
4. **JWT fulfills its real purpose** — protecting the analytics dashboard. The spec requirement lives there.

The tracking ID is not a security boundary. Server-side guardrails handle abuse:

- Rate limiting per tracking ID / IP
- Schema validation (reject malformed events)
- Event type allowlist
- Anomaly detection (e.g., 10K purchase events in 1 second from one origin)
- Option to disable event ingestion per customer via settings

### Decision Record

| Criteria | Strategy A (Public ID) | Strategy B (Body Token) | Strategy C (Shared JWT) | Strategy D (Two-Tier) |
|---|---|---|---|---|
| Works with `sendBeacon` | ✅ Yes | ✅ Yes | ⚠️ JWT timeouts silently drop events | ✅ Yes |
| Simple customer setup | ✅ Copy-paste snippet | ✅ Copy-paste snippet | ❌ Customer needs a BFF | ✅ Copy-paste snippet |
| Prevents spoofing | ❌ No | ❌ No (token is visible) | ⚠️ Binds to user session only | ❌ No |
| Industry precedent | ✅ GA, Mixpanel, Amplitude | ⚠️ Segment (write key, also public) | ❌ No major product | ❌ No |
| Multi-tenant | ✅ Yes | ✅ Yes | ❌ Single tenant | ✅ Yes |
| Spec alignment (JWT) | ✅ JWT on analytics | ✅ JWT on analytics | ✅ JWT on both (but broken model) | ✅ JWT on analytics |

**Conclusion**: Strategy A. The tracking ID identifies the customer. JWT protects the analytics dashboard. The spec's "JWT in HttpOnly cookies" applies to dashboard auth — the most meaningful place to authenticate.

---

## Implementation

### API: JWT auth for analytics

```
POST /api/auth/signup  — create account, returns tracking ID
POST /api/auth/login   — sign in, sets JWT HttpOnly cookie
POST /api/auth/logout  — clears cookie
GET  /api/analytics    — requires valid JWT cookie
GET  /health           — no auth
GET  /metrics          — no auth (prometheus scraper)
```

### Event ingestion

```
POST /track/events — no auth, accepts {tracking_id, events: [...]}
```

The tracking script sends a tracking ID from `window._TRACKER_ID`. The API validates the ID exists, scopes events to that customer, and persists them.

### Execution

- Add JWT helpers to the API package (sign, verify, middleware) — analogous to `ecommerce/auth.go`
- Add `/api/auth/*` routes to the API server
- Add `tracking_id` column to the ClickHouse schema
- Add tracking ID validation/tenant isolation to `Store.ProcessEvents`
- Add JWT auth middleware to `GET /api/analytics`
- Add signup (creates account + generates tracking ID) and login endpoints
