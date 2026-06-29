# E-Commerce App — Design Document

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                    Browser                            │
│  ┌──────────────────────────────────────────────┐   │
│  │  SPA (ecommerce/app/)                         │   │
│  │  React 19 + React Router 7                    │   │
│  │  Zustand + Tailwind CSS 4 + shadcn/ui         │   │
│  │                                                │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────────┐  │   │
│  │  │ Auth     │ │ Products │ │ Cart/Checkout │  │   │
│  │  │ Pages    │ │ Page     │ │ Pages         │  │   │
│  │  └────┬─────┘ └────┬─────┘ └──────┬───────┘  │   │
│  │       │            │              │           │   │
│  │       └────────────┼──────────────┘           │   │
│  │                    │                          │   │
│  │               ┌────▼─────┐                    │   │
│  │               │ lib/api  │                    │   │
│  │               │ (fetch)  │                    │   │
│  │               └────┬─────┘                    │   │
│  └────────────────────┼──────────────────────────┘   │
└───────────────────────┼──────────────────────────────┘
                        │ same-origin
                        ▼
┌─────────────────────────────────────────────────────┐
│  BFF — Go (ecommerce/)                              │
│                                                      │
│  main.go (Gin :8080)                                 │
│  ┌──────────────┐  ┌──────────────┐                  │
│  │ Auth         │  │ Static       │                  │
│  │ /api/auth/*  │  │ /app (dist)  │                  │
│  │ JWT HttpOnly │  │ SPA fallback │                  │
│  └──────────────┘  └──────────────┘                  │
│  ┌──────────────┐                                    │
│  │ Products API │                                    │
│  │ /api/products│                                    │
│  └──────────────┘                                    │
└─────────────────────────────────────────────────────┘
```

The BFF and SPA are deployed as a single unit. The Go binary embeds the built frontend or serves it from `app/dist/`. No reverse proxy needed — same origin means cookies flow naturally.

---

## Directory Structure

```
ecommerce/
├── main.go              # Entry point, routes, static serving
├── auth.go              # JWT + user store + auth handlers
├── go.mod / go.sum
├── products.json        # Synthetic product data (embedded)
├── DESIGN.md
├── tests/               # Playwright E2E tests
└── app/                 # Vite+ React SPA
    ├── index.html
    ├── vite.config.js   # base: '/app/', Tailwind, alias
    ├── package.json     # deps + test:e2e script
    └── src/
        ├── main.jsx     # Entry point
        ├── App.jsx      # Router + layout
        ├── index.css    # Tailwind + shadcn CSS variables
        ├── components/  # Navbar, ErrorBoundary, ui/ (shadcn)
        ├── pages/       # Login, Signup, Products, ProductDetail, Cart, Checkout
        ├── stores/      # authStore, cartStore (Zustand)
        └── lib/         # api.js, logger.js, utils.js
```

---

## Routes

### Frontend (React Router 7)

| Path | Component | Description |
|------|-----------|-------------|
| `/login` | `Login` | Sign in form, redirects to /products on success |
| `/signup` | `Signup` | Registration form, redirects to /products on success |
| `/products` | `Products` | Product grid, fetches from BFF API |
| `/products/:id` | `ProductDetail` | Full product view with qty selector |
| `/cart` | `Cart` | Line items, quantity controls, order summary |
| `/checkout` | `Checkout` | Multi-step (Shipping → Payment → Review → Place) |
| `*` | redirect → `/products` | Catch-all SPA fallback |

### BFF API (Gin)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/auth/signup` | No | Create user, set JWT cookie |
| POST | `/api/auth/login` | No | Validate credentials, set JWT cookie |
| POST | `/api/auth/logout` | No | Clear JWT cookie |
| GET | `/api/auth/me` | Yes | Return `{user}` from JWT claims |
| GET | `/api/products` | No | Return all products (synthetic JSON) |
| GET | `/api/products/:id` | No | Return single product by ID |
| GET | `/health` | No | `{"status":"ok"}` |

---

## State Management (Zustand)

### authStore
- `user: {email, name} | null` — set after login/signup, loaded from localStorage for persistence across refreshes
- `isLoading`, `error` — for UI states
- `login(email, password)` → `POST /api/auth/login` → stores user, sets no token in JS (HttpOnly cookie handles it)
- `signup(name, email, password)` → `POST /api/auth/signup`
- `logout()` → `POST /api/auth/logout` → clears user
- `checkAuth()` → `GET /api/auth/me` — restores session on page load via cookie

### cartStore
- `items: [{product, quantity}]` — persisted to `localStorage`
- `addItem(product, qty)` — increments quantity if already in cart
- `removeItem(productId)` — deletes item
- `updateQuantity(productId, qty)` — removes if qty ≤ 0
- `clearCart()` — empties after successful checkout

**Decision: Zustand over Context/Redux.** Zustand has zero boilerplate, works outside React components, and selectors prevent unnecessary re-renders. Two small stores are clearer than one monolithic store or a deeply nested Context.

---

## Authentication Flow

### JWT (Custom Implementation)
- Header: `{"alg":"HS256","typ":"JWT"}`
- Payload: `{email, name, exp}` (24-hour expiry)
- Signature: HMAC-SHA256 of `base64(header).base64(payload)`
- Key: hardcoded in `auth.go` (env var in production)
- No JWT library — ~40 lines of stdlib (`crypto/hmac`, `crypto/sha256`, `encoding/base64`, `encoding/json`)

### Cookie
- `Set-Cookie: token=<jwt>; HttpOnly; Secure; SameSite=Lax; Path=/; Max-Age=86400`
- **HttpOnly** — JavaScript cannot read the token; XSS cannot steal it
- **Secure** — only sent over HTTPS (localhost exempted by browser)
- **SameSite=Lax** — prevents CSRF from external sites while allowing same-origin navigation
- 24-hour expiry, cleared on logout via `Max-Age=0`

### Why not localStorage?
- localStorage is accessible to any JavaScript on the same origin — one XSS vulnerability leaks the token permanently
- HttpOnly cookies are invisible to JS; the browser handles them automatically
- Trade-off: requires same-origin deployment (BFF + SPA on one domain) or careful CORS + cookie configuration for cross-origin

### User Store
- In-memory `map[string]User` protected by `sync.RWMutex`
- Passwords stored in plaintext — intentional for demo scope
- Production: bcrypt/argon2 + database

---

## Frontend Build + Serving

### Build Pipeline
- `vp build` (Vite+): outputs to `app/dist/` as a standard SPA (no `client/`/`server/` subdirectories)
- `base: '/app/'` in `vite.config.js` — all asset URLs prefixed with `/app`
- Single `index.html` entry point for all routes

### BFF Serving
- `r.Static("/app", "app/dist")` serves static assets
- `NoRoute` handler serves `app/dist/index.html` for all unmatched paths — enables client-side routing with direct URL access
- Root `/` redirects to `/app/`

---

## Key Decisions

| # | Decision | Rationale |
|---|----------|-----------|
| 1 | SPA mode (no SSR) | No Node.js runtime needed; Go serves static files; simpler deploy |
| 2 | `base: '/app/'` | Separates API routes (`/api/*`) from frontend routes; clean namespace |
| 3 | Custom JWT over library | ~40 lines of stdlib vs pulling `golang-jwt/jwt`; adequate for demo |
| 4 | In-memory user store | Zero dependencies; resets on restart which is acceptable for demo |
| 5 | Cart in Zustand + localStorage | Survives page refresh; no server needed for cart state |
| 6 | Products served from BFF | Same-origin API calls; no CORS issues; single deployable unit |
| 7 | Multi-step checkout (4 steps) | Spec-required moderately complex workflow; accumulated local state |
| 8 | ErrorBoundary + logger | Catches render crashes; structured logs for debugging |
| 9 | shadcn/ui with Tailwind 4 | Consistent design system; accessible out of the box |
| 10 | Fast-food color palette | (McDonald's-inspired red/yellow) Distinctive brand identity for the demo |

---

## Component Tree

```
<App>
  <BrowserRouter>
    <Navbar />
    <ErrorBoundary>
      <Routes>
        <Login />
        <Signup />
        <Products />
        <ProductDetail />
        <Cart />
        <Checkout>
          <StepIndicator />
          <ShippingStep />
          <PaymentStep />
          <ReviewStep />
          <ConfirmationStep />
        </Checkout>
      </Routes>
    </ErrorBoundary>
  </BrowserRouter>
</App>
```

---

## Data Flow

### Page: Login
```
User fills form → validate() → authStore.login(email, pass)
  → api.login() → fetch POST /api/auth/login
  → BFF: handleLogin → check password → createToken → setCookie
  → Response: {user} → localStorage.setItem + setState
  → navigate(/products)
```

### Page: Products
```
mount → load() → fetch GET /api/products
  → BFF: embedded products.json → return JSON
  → setProducts(data)
  
User clicks "Add to Cart" → cartStore.addItem(product)
  → update localStorage → badge updates via selector
```

### Page: Checkout
```
Step 1: Shipping form → validate → next
Step 2: Payment form → validate → next
Step 3: Review (read-only summary) → "Place Order"
  → simulate 2s delay → clearCart → navigate(/products)
```

---

## Styling Approach

- **Tailwind CSS 4**: utility classes for layout, spacing, typography
- **shadcn/ui**: accessible primitives (Button, Card, Input, Badge, etc.) with consistent API
- **CSS variables**: color tokens defined in `index.css`; light/dark mode via `.dark` class
- **Custom theme**: McDonald's-inspired palette (red primary `#da291c`, yellow accent `#ffc72c`, warm cream background)

---

## Accessibility

- Semantic HTML: `<main>`, `<header>`, `<nav>`, `<form>`, `<h1>`–`<h2>`
- Labels: all form inputs use `<Label htmlFor="id">` + `id` on `<Input>`
- Error associations: `aria-invalid`, `aria-describedby` linking inputs to error messages
- ARIA: `aria-label` on icon buttons, `aria-current="step"` on checkout progress, `aria-live="polite"` on dynamic content
- Keyboard: all interactive elements are `<button>` or `<a>` (natively keyboardable)
- Focus: `focus-visible` ring styles on all interactive elements

---

## Error Handling Strategy

| Layer | Approach |
|-------|----------|
| API calls | `try/catch` in store actions; `error` state displayed as inline alert |
| Form validation | Field-level errors via `fieldErrors` state; no invalid submits |
| Product fetch | Loading skeleton → error state with retry button → empty state |
| Product detail | Loading skeleton → not-found → error with back link |
| Cart empty | Dedicated empty state with CTA to browse products |
| Render errors | `<ErrorBoundary>` wraps all routes; logs via `logger.error`; shows recovery UI |
| Network errors | `api.js` throws on non-OK responses; stores catch and surface `error.message` |

---

## Testing Strategy

| Type | Scope | Tools |
|------|-------|-------|
| Unit | Zustand stores, utility functions (`logger`, `api`) | Vitest |
| Integration | Page renders, form submissions, data fetching | Vitest + React Testing Library |
| E2E | Full flows: signup → login → browse → add to cart → checkout | Playwright |

Key flows to test:
1. Signup with valid/invalid data → assert redirect on success, error on duplicate
2. Login with valid/invalid credentials → assert cookie set / error shown
3. Products load and display → assert grid renders with fetched items
4. Add to cart → assert badge count increments, cart page shows item
5. Complete checkout → assert items cleared, redirect to products
6. Unauthenticated /me call → assert 401 handled gracefully

---

## How to Run

```sh
# Install frontend deps
cd ecommerce/app && pnpm install

# Build frontend
pnpm build

# Start BFF (from repo root or ecommerce/)
cd .. && go run ./ecommerce

# Open http://localhost:8080
```
