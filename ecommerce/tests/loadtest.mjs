const API = process.env.API_URL || "http://localhost:8081";
const TOTAL = parseInt(process.env.TOTAL_EVENTS || "2000", 10);
const BATCH = 100;
const PURCHASE_RATIO = 0.25;
const DEDUP_RATIO = 0.1;

const TYPES = ["page_view", "click", "add_to_cart", "checkout", "payment_info", "lead", "purchase"];

const PRODUCTS = [
  { id: 1, title: "Wireless Headphones", price: 79.99, category: "Electronics" },
  { id: 2, title: "Running Shoes", price: 129.99, category: "Sports" },
  { id: 3, title: "Coffee Maker", price: 49.99, category: "Home" },
  { id: 4, title: "Backpack", price: 89.99, category: "Accessories" },
  { id: 5, title: "Desk Lamp", price: 39.99, category: "Home" },
  { id: 6, title: "Yoga Mat", price: 29.99, category: "Sports" },
  { id: 7, title: "Wireless Mouse", price: 24.99, category: "Electronics" },
  { id: 8, title: "Water Bottle", price: 19.99, category: "Sports" },
  { id: 9, title: "Notebook", price: 9.99, category: "Stationery" },
  { id: 10, title: "Sunglasses", price: 59.99, category: "Accessories" },
];

const PATHS = ["/", "/products", "/products/1", "/products/2", "/cart", "/checkout", "/login", "/signup"];
const TIMEZONES = ["America/New_York", "Europe/London", "Asia/Tokyo", "UTC"];
const LOCATIONS = ["localhost", "staging.example.com"];
const PAYMENT_METHODS = ["card", "paypal", "apple_pay"];

function pick(arr) { return arr[Math.floor(Math.random() * arr.length)]; }
function randInt(min, max) { return Math.floor(Math.random() * (max - min + 1)) + min; }

function randomItems() {
  const count = randInt(1, 4);
  const used = new Set();
  const items = [];
  for (let i = 0; i < count; i++) {
    let p;
    do { p = pick(PRODUCTS); } while (used.has(p.id));
    used.add(p.id);
    items.push({ product_id: p.id, title: p.title, price: p.price, quantity: randInt(1, 3) });
  }
  return items;
}

const purchaseIds = [];
let dedupTargets = [];

const events = [];

for (let i = 0; i < TOTAL; i++) {
  const isPurchase = Math.random() < PURCHASE_RATIO;
  const type = isPurchase ? "purchase" : pick(TYPES.filter(t => t !== "purchase"));

  let id;
  let data;

  if (type === "purchase") {
    if (Math.random() < DEDUP_RATIO && dedupTargets.length > 0) {
      id = pick(dedupTargets);
    } else {
      id = crypto.randomUUID();
      purchaseIds.push(id);
      if (purchaseIds.length >= 5 && Math.random() < 0.5) {
        dedupTargets.push(id);
      }
    }
    const items = randomItems();
    data = { order_id: `ORD-LD-${String(i).padStart(5, "0")}`, items, total: +items.reduce((s, it) => s + it.price * it.quantity, 0).toFixed(2) };
  } else if (type === "page_view") {
    id = crypto.randomUUID();
    data = { path: pick(PATHS), referrer: Math.random() < 0.3 ? "https://google.com" : "" };
  } else if (type === "click") {
    id = crypto.randomUUID();
    const tag = pick(["A", "BUTTON"]);
    data = { tag, id: "", class: pick(["nav-link", "btn-primary", "btn-secondary"]), text: "click", href: tag === "A" ? pick(PATHS) : "" };
  } else if (type === "add_to_cart") {
    id = crypto.randomUUID();
    const p = pick(PRODUCTS);
    data = { product_id: p.id, product_title: p.title, price: p.price, quantity: randInt(1, 3) };
  } else if (type === "checkout") {
    id = crypto.randomUUID();
    const items = randomItems();
    data = { items, total: +items.reduce((s, it) => s + it.price * it.quantity, 0).toFixed(2) };
  } else if (type === "payment_info") {
    id = crypto.randomUUID();
    data = { method: pick(PAYMENT_METHODS) };
  } else if (type === "lead") {
    id = crypto.randomUUID();
    data = { email: `user${i}@example.com`, name: `User ${i}` };
  }

  events.push({
    id,
    type,
    timestamp: new Date(Date.now() - randInt(0, 7 * 86400000)).toISOString(),
    source: "loadtest",
    origin: "https://loadtest.example.com",
    session_id: crypto.randomUUID(),
    user_agent: "LoadTest/1.0",
    timezone: pick(TIMEZONES),
    location: pick(LOCATIONS),
    url: type === "page_view" ? `http://localhost:8080${data.path}` : "http://localhost:8080/",
    data,
  });
}

console.log(`\nTarget API: ${API}`);
console.log(`Total events: ${TOTAL}`);

const stats = { purchase: 0, page_view: 0, click: 0, add_to_cart: 0, checkout: 0, payment_info: 0, lead: 0 };
for (const e of events) { stats[e.type]++; }
console.log("\n=== Generated Events ===");
for (const [t, c] of Object.entries(stats)) {
  console.log(`  ${t.padEnd(14)} ${String(c).padStart(5)} (${(c / TOTAL * 100).toFixed(1)}%)`);
}

const uniquePurchaseIds = new Set(events.filter(e => e.type === "purchase").map(e => e.id));
console.log(`  ${"purchase (unique)".padEnd(14)} ${String(uniquePurchaseIds.size).padStart(5)} (after dedup)`);
const expectedDedupDrops = stats.purchase - uniquePurchaseIds.size;

// Send batches
let sent = 0, processed = 0, dropped = 0;
const counts = {};

console.log(`\nSending ${events.length} events in ${Math.ceil(events.length / BATCH)} batches...`);

for (let i = 0; i < events.length; i += BATCH) {
  const batch = events.slice(i, i + BATCH);
  const res = await fetch(`${API}/track/events`, {
    method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(batch),
  });
  const body = await res.json();
  sent += body.received;
  processed += body.processed;
  dropped += body.dropped;
  for (const m of body.metadata) {
    counts[m.Name] ??= { processed: 0, errors: 0, dropped: 0, latencyNs: 0, throughput: 0 };
    counts[m.Name].processed += m.Processed;
    counts[m.Name].errors += m.Errors;
    counts[m.Name].dropped += m.Dropped;
    counts[m.Name].latencyNs += m.LatencyNs;
    counts[m.Name].throughput = m.Throughput;
  }
  process.stdout.write(`\r  batch ${Math.floor(i / BATCH) + 1}/${Math.ceil(events.length / BATCH)}: sent=${body.received} proc=${body.processed} drop=${body.dropped}`);
}

console.log("\n\n=== Ingestion ===");
console.log(`  Events sent:     ${sent}`);
console.log(`  Processed:       ${processed}`);
console.log(`  Dropped:         ${dropped}`);
console.log(`\n  Pipeline stage breakdown:`);
for (const [name, c] of Object.entries(counts)) {
  console.log(`    ${name.padEnd(16)} processed=${String(c.processed).padStart(5)} errors=${c.errors} dropped=${String(c.dropped).padStart(5)}`);
}

// Wait for ClickHouse flush
console.log("\nWaiting for ClickHouse async flush...");
await new Promise(r => setTimeout(r, 6000));

// Auth
console.log("Authenticating...");
await fetch(`${API}/api/auth/signup`, {
  method: "POST", headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ name: "Load Tester", email: "loadtest@example.com", password: "test123" }),
});
const loginRes = await fetch(`${API}/api/auth/login`, {
  method: "POST", headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ email: "loadtest@example.com", password: "test123" }),
});
const cookies = loginRes.headers.getSetCookie().join("; ");

// Check analytics
console.log("Fetching analytics...");
const analyticsRes = await fetch(`${API}/api/analytics`, { headers: { Cookie: cookies } });
const analytics = await analyticsRes.json();

console.log("\n=== Analytics ===");
console.log(JSON.stringify(analytics, null, 2));

// Verify
console.log("\n=== Verification ===");
const expectedUniquePurchases = Math.round(stats.purchase * (1 - DEDUP_RATIO));
const dbTotal = analytics.total_events;
const dbByType = analytics.events_by_type;
let failures = 0;

// 1. Pipeline metadata has the expected 4 stages
console.log(`  Pipeline stages:                  ${Object.keys(counts).length} (expected 4)`);
const expectedStages = ["validate", "normalize", "dedup", "enrichIfPurchase"];
for (const name of expectedStages) {
  if (!counts[name]) {
    console.log(`    ✗ ${name} missing`);
    failures++;
  }
}

// 2. Dedup drops match expected
const dedupDrops = counts.dedup?.dropped ?? 0;
console.log(`  Dedup drops:                      ${dedupDrops} (expected ~${expectedDedupDrops})`);
const dedupDiff = Math.abs(dedupDrops - expectedDedupDrops);
if (dedupDiff > Math.max(1, expectedDedupDrops * 0.1)) {
  console.log(`    ✗ dedup drop count off by ${dedupDiff}`);
  failures++;
}

// 3. DB total events matches processed count
console.log(`  Events in DB:                     ${dbTotal}`);
console.log(`  Events processed:                 ${processed}`);
if (dbTotal !== processed) {
  console.log(`    ✗ DB total (${dbTotal}) ≠ processed (${processed})`);
  failures++;
}

// 4. Events by type — expect all 7 types present
const typesInDB = Object.keys(dbByType).length;
console.log(`  Event types in DB:                ${typesInDB} (expected 7)`);
if (typesInDB !== 7) {
  console.log(`    ✗ expected 7 types, got ${typesInDB}`);
  failures++;
}

// 5. Pipeline metadata latency/throughput present
for (const name of expectedStages) {
  const lat = counts[name]?.latencyNs ?? 0;
  const thr = counts[name]?.throughput ?? 0;
  if (lat === 0) {
    console.log(`    ${name}: latency=0 — expected >0`);
  }
  if (thr === 0) {
    console.log(`    ${name}: throughput=0 — expected >0`);
  }
}

console.log(`\n  Avg capture time (ms):           ${analytics.avg_capture_time_ms.toFixed(0)}`);
console.log(`  Avg event params:                ${analytics.avg_event_params.toFixed(2)}`);

if (failures === 0) {
  console.log("\n✅ All verifications passed!");
} else {
  console.log(`\n❌ ${failures} verification(s) failed`);
}

console.log("\n✅ Load test complete!");
