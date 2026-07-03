# Pipeline Library — Design Document

## Overview

Generic, zero-dependency Go streaming pipeline library. Stages are chained via buffered channels, each running in its own goroutine. Per-stage atomic counters track processed/errors/dropped. No external dependencies — stdlib only.

## Architecture

```
          in ──► Stage 1 ──► Stage 2 ──► ... ──► Stage N ──► out
                 │            │                      │
               Meta()       Meta()                 Meta()
               (atomic)     (atomic)               (atomic)
```

Each `Stage[T]` implements `Process(ctx, in, out)` and `Meta() StageMeta`. `Pipeline.Run(ctx, in)` chains them into goroutines and returns a metadata closure that reports per-run deltas (snapshot taken before run starts, subtracted from current counters when the closure is called).

## Stage Types

| Type | Input | Output | Behaviour |
|------|-------|--------|-----------|
| `Map` | T | T | 1:1 transform; error → counted and skipped |
| `Filter` | T | T | Predicate decides keep/drop; dropped items counted |
| `Deduplicate` | T | T | In-memory seen-set by keyFn; drops duplicates by ID |
| `If` | T | T | Branch on condition — one transform per branch |
| `Generate` | (none) | T | Source stage; produces items via emit callback |
| `ChurnEnrich` | T (ChurnEnrichable) | T | Assigns random churn score; demo stage |
| `Collect` | T | []T | Batches items into slices of batchSize |
| `Reduce` | T | R | Folds items into accumulator; standalone function, not a Stage |
| `FanOut` | T | T | Spawns N goroutines sharing one Stage against the same input; results merged into a single output channel |

## Meta Contract

`StageMeta` contains `Name`, `Processed`, `Errors`, `Dropped`. Counters are on `Core` (embedded atomic.Int64). The metadata closure returned by `Run()` computes deltas from a pre-run snapshot, so repeated calls on the same pipeline report only each run's contributions. Call `s.Meta()` directly on a stage for cumulative totals.

## Deduplicate Details

In-memory `map[string]struct{}` with configurable capacity. When the map reaches capacity it is reset (allows a fresh set of ~cap keys). Only in-process duplication is tracked — no persistent cross-run dedup.

## ChurnEnrich

Constraint-based generic stage requiring `SetChurnScore(ChurnScore)` on the item. Assigns `rand.Float64()` as the probability and `"simulated"` as the reason. No real ML — purely a demo stage to show the generic plumbing works.

## Why Not

- **No error propagation upstream** — errors log on the stage and the item is skipped. Pipeline continues.
- **No backpressure** — channel buffers absorb bursts; full channel blocks the producer. FanOut can help spread load.
- **No serialization** — items stay in memory as Go values. The library is agnostic about encoding.

## Dependencies

Zero. `go.mod` declares only the module path and Go version. No third-party packages.
