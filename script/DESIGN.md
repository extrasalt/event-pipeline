# Tracking Script — Design Document

## Overview

Vanilla JS IIFE that auto-instruments 7 event types with zero SPA code changes. Loaded from the HTML `<head>` via a `<script src="/app/script/tracker.js">` tag. No build step, no npm, no framework dependency.

## Event Types

| Event | Trigger | Data |
|-------|---------|------|
| `page_view` | DOM load + SPA navigation (history API monkey-patch) | path, referrer |
| `click` | Global click listener on `a, button, [role=button]` | tag, id, class, text, href |
| `add_to_cart` | `localStorage.setItem("cart")` interception, diffed against previous state | product_id, title, price, quantity |
| `checkout` | URL path matches `/checkout` | items, total |
| `payment_info` | Payment form fields detected in DOM (waitForElement) | method |
| `purchase` | Cart goes empty after checkout + confirmation text detected (waitForText) | order_id, items, total |
| `lead` | Form submit detects email + name inputs | email |

## Session

- Session ID generated once via `crypto.randomUUID()`, stored in `sessionStorage._tsid`. Persists for the tab session.
- Cart state tracked via `localStorage.getItem("cart")` on load, then monkey-patched `Storage.prototype.setItem` to detect changes.
- One-shot guards (`_checkoutFired`, `_paymentFired`, `_purchaseFired`) prevent duplicate events per session.

## Transport

- `navigator.sendBeacon(API_URL, body)` — non-blocking, survives page unload. Falls back to `fetch(..., {keepalive: true})` if sendBeacon is unavailable.
- No response handling — fire and forget.
- API URL configurable via `window._TRACKER_API` (defaults to `http://localhost:8081/track/events`).

## Guard Pattern

```js
if (window.__tracker) return;
```
Prevents double initialization. Exposes `window.__tracker.send()` and `window.__tracker.pageView()` for manual use if needed.

## DOM Detection

- `waitForElement(selector, timeout, callback)` — polls `document.querySelector` until the element appears (used for payment form fields).
- `waitForText(pattern, timeout, callback)` — polls `document.body.innerText` for a regex pattern (used for order confirmation text after purchase).

## Why Vanilla JS IIFE

- No build step, no npm, no bundler — one file, drop it in.
- Works with any SPA framework or no framework at all.
- Under 3KB minified.
- CDN-deployable independently.
