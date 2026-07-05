# Benchmarks

## Environment

| Spec | Value |
|------|-------|
| CPU | Apple M1 |
| Cores | 8 (4 performance + 4 efficiency) |
| RAM | 8 GB unified |
| Go version | 1.25 |
| OS | macOS (Darwin) |

## Methodology

Benchmarks defined in `pipeline/pipeline_test.go`. Two configurations:

- **`BenchmarkPipeline`**: `int → Map(id) → Filter(even)`. Baseline throughput — measures channel overhead and stage chaining without allocation-heavy operations.
- **`BenchmarkPipelineDedupChurn`**: `testEvent → validate → filter purchase only → dedup → churn enrich`. Mirrors the production event pipeline. Every third event is `"click"` (filtered out), so ~67% of events survive to output.

Both benchmarks run at four event volumes: 10, 1,000, 100,000, and 1,000,000. Each volume is a sub-benchmark that sends all events through a single `Pipeline.Run` call. `b.ReportAllocs()` is enabled. `benchtime=10ms` (Go default for longer iterations).

Run with:

```
go test -bench=. -benchtime=10ms ./pipeline/
```

## Results

```
BenchmarkPipeline/N=10-8         	    1488	      7291 ns/op	   28777 B/op	       8 allocs/op
BenchmarkPipeline/N=1000-8      	      22	    618371 ns/op	   28744 B/op	       8 allocs/op
BenchmarkPipeline/N=100000-8    	       1	  33316542 ns/op	   28760 B/op	       9 allocs/op
BenchmarkPipeline/N=1000000-8   	       1	 235732292 ns/op	   28744 B/op	       8 allocs/op
BenchmarkPipelineDedupChurn/N=10-8         	      52	    283047 ns/op	 3545225 B/op	     297 allocs/op
BenchmarkPipelineDedupChurn/N=1000-8       	      15	    765708 ns/op	 3606878 B/op	    3022 allocs/op
BenchmarkPipelineDedupChurn/N=100000-8     	       1	  36767250 ns/op	10668080 B/op	  300063 allocs/op
BenchmarkPipelineDedupChurn/N=1000000-8    	       1	 348256291 ns/op	96480824 B/op	 3001847 allocs/op
```

## Analysis

The dedup map is the bottleneck — it keeps a seen-set that grows with unique event count. Default capacity (10K) keeps memory reasonable; at 1M events the map resets once, keeping allocations at ~3M for the full pipeline. The baseline pipeline (no dedup) processes 1M ints through Map + Filter in ~236ms with negligible allocations. The full pipeline (validation + dedup + churn enrichment) takes ~348ms for 1M events — dominated by the dedup seen-set and per-event allocation overhead.
