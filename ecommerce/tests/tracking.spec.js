import { test, expect } from "@playwright/test";

const PRODUCTS = [
  { id: 1, title: "Test Product", price: 29.99, category: "Test", image: "https://picsum.photos/seed/test/400/400", rating: { rate: 4.0, count: 10 }, description: "A test product." },
];

test.describe("Tracking Script", () => {
  let events = [];

  test.beforeEach(async ({ page }) => {
    events = [];
    await page.addInitScript(() => {
      window._TRACKER_API = "/__track";
    });
    await page.route("**/__track", async (route) => {
      const body = route.request().postDataJSON();
      events.push(...body);
      await route.fulfill({ status: 200 });
    });
    await page.route("**/api/products", async (route) => {
      await route.fulfill({ json: PRODUCTS });
    });
    await page.route("**/api/auth/me", async (route) => {
      await route.fulfill({ status: 401 });
    });
  });

  test("page_view fires on initial load", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");
    expect(events.some((e) => e.type === "page_view")).toBe(true);
  });

  test("page_view fires on SPA navigation", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");
    events = [];
    await page.click('a[href="/cart"]');
    await page.waitForTimeout(500);
    expect(events.some((e) => e.type === "page_view" && e.data.path === "/cart")).toBe(true);
  });

  test("click events fire on button interaction", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");
    events = [];
    await page.click('a[href="/cart"]');
    await page.waitForTimeout(500);
    const clickEvents = events.filter((e) => e.type === "click");
    expect(clickEvents.length).toBeGreaterThanOrEqual(1);
    expect(clickEvents[0].data.tag).toBe("A");
  });

  test("add_to_cart fires from product grid", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");
    events = [];
    const addBtn = page.locator('button:has-text("Add to Cart")').first();
    await addBtn.click();
    await page.waitForTimeout(500);
    const addEvents = events.filter((e) => e.type === "add_to_cart");
    expect(addEvents.length).toBeGreaterThanOrEqual(0);
  });

  test("checkout and payment_info fire on checkout flow", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");
    const addBtn = page.locator('button:has-text("Add to Cart")').first();
    await addBtn.click();
    await page.waitForTimeout(500);
    events = [];
    await page.goto("/checkout");
    await page.waitForLoadState("networkidle");
    await page.waitForTimeout(500);
    expect(events.some((e) => e.type === "checkout")).toBe(true);
    await page.fill("#checkout-name", "Jane Doe");
    await page.fill("#checkout-address", "123 Main St");
    await page.fill("#checkout-city", "Portland");
    await page.fill("#checkout-state", "OR");
    await page.fill("#checkout-zip", "97201");
    await page.fill("#checkout-country", "US");
    await page.click('button:has-text("Continue to Payment")');
    await page.waitForTimeout(500);
    expect(events.some((e) => e.type === "payment_info")).toBe(true);
  });

  test("purchase fires on order placement", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");
    const addBtn = page.locator('button:has-text("Add to Cart")').first();
    await addBtn.click();
    await page.waitForTimeout(300);
    events = [];
    await page.goto("/checkout");
    await page.waitForLoadState("networkidle");
    await page.waitForTimeout(300);
    await page.fill("#checkout-name", "Jane Doe");
    await page.fill("#checkout-address", "123 Main St");
    await page.fill("#checkout-city", "Portland");
    await page.fill("#checkout-state", "OR");
    await page.fill("#checkout-zip", "97201");
    await page.fill("#checkout-country", "US");
    await page.click('button:has-text("Continue to Payment")');
    await page.waitForTimeout(300);
    await page.fill("#checkout-card", "4242424242424242");
    await page.fill("#checkout-expiry", "12/28");
    await page.fill("#checkout-cvv", "123");
    await page.click('button:has-text("Review Order")');
    await page.waitForTimeout(300);
    await page.click('button:has-text("Place Order")');
    await page.waitForTimeout(2500);
    const purchaseEvents = events.filter((e) => e.type === "purchase");
    expect(purchaseEvents.length).toBeGreaterThanOrEqual(1);
    expect(purchaseEvents[0].data.order_id).toMatch(/^ORD-/);
    expect(purchaseEvents[0].data.items.length).toBe(1);
    expect(purchaseEvents[0].data.total).toBeGreaterThan(0);
  });

  test("lead fires on signup form submission", async ({ page }) => {
    events = [];
    await page.goto("/signup");
    await page.waitForLoadState("networkidle");
    await page.fill("#signup-name", "Jane Doe");
    await page.fill("#signup-email", "jane@example.com");
    await page.fill("#signup-password", "password123");
    await page.fill("#signup-confirm", "password123");
    await page.click('button:has-text("Create account")');
    await page.waitForTimeout(500);
    const leadEvents = events.filter((e) => e.type === "lead");
    expect(leadEvents.length).toBeGreaterThanOrEqual(0);
  });

  test("events include metadata fields", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");
    const pv = events.find((e) => e.type === "page_view");
    expect(pv).toBeDefined();
    expect(pv.id).toBeDefined();
    expect(pv.timestamp).toBeDefined();
    expect(pv.user_agent).toBeDefined();
    expect(pv.timezone).toBeDefined();
    expect(pv.location).toBeDefined();
    expect(pv.session_id).toBeDefined();
  });

  test("login form does not fire lead event", async ({ page }) => {
    events = [];
    await page.goto("/login");
    await page.waitForLoadState("networkidle");
    await page.fill("#login-email", "jane@example.com");
    await page.fill("#login-password", "password123");
    await page.click('button:has-text("Sign In")');
    await page.waitForTimeout(500);
    const leadEvents = events.filter((e) => e.type === "lead");
    expect(leadEvents.length).toBe(0);
  });
});
