# ClickHouse vs SQLite + Litestream

## Context

We need an analytics store for the event tracking pipeline. Every event that passes through the pipeline gets persisted, and the API serves aggregate analytics (counts by type, average capture time, events over time).

Two candidate approaches: ClickHouse (column-oriented OLAP) and SQLite with Litestream streaming S3 backups.

## Why Not SQLite

SQLite is a great embedded OLTP database. For our workload it is the wrong tool.

The API queries are analytical — scans over time ranges by event type. SQLite stores rows as contiguous pages. A query like `SELECT type, count(*) FROM events WHERE timestamp > now() - INTERVAL 1 DAY GROUP BY type` reads every row in the range, parses each page, and filters in the query engine. As the event table grows into millions of rows, these scans get progressively slower.

SQLite serializes writes behind a single mutex. Our pipeline can burst thousands of events per second. That mutex becomes a bottleneck before anything else.

Litestream solves the durability problem — continuous S3 backup is a good story for disaster recovery. It does not change the query model, the single-writer lock, or the row-scan performance ceiling.

## Why ClickHouse

ClickHouse is purpose-built for exactly this workload.

- Columnar storage means aggregate queries read only the columns they need, not entire rows.
- Vectorized execution compiles aggregation pipelines into SIMD-friendly operations.
- Real-time inserts handle millions of rows per second across multiple shards.
- Built-in time-series functions give us events-over-time as a single SQL expression, not application-level bucketing.

The operational cost is real — it runs as a separate service with its own config, ZK/Keeper for replication, and the SQL dialect has sharp edges. But for a production event pipeline where the primary queries are time-range aggregates, the query engine matters more than deployment simplicity.

## Verdict

SQLite + Litestream is a reasonable choice if the event volume stays under ~100K rows and operational simplicity is the top concern. For this system — where we expect sustained throughput and the API surface is aggregates over time — ClickHouse is the correct default.
