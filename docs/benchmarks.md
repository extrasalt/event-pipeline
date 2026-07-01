# Benchmarks

From `pipeline/`, run:

```
go test -bench=. -benchtime=10ms
```

Two configurations:

- **`BenchmarkPipeline`**: int → Map(id) → Filter(even). Baseline throughput.
- **`BenchmarkPipelineDedupChurn`**: testEvent → validate → filter purchase only → dedup → churn enrich. Matches production. Every third event is "click" (filtered out), ~67% survive.

```
BenchmarkPipeline/N=10-8         	     322	     45606 ns/op	   28663 B/op	       7 allocs/op
BenchmarkPipeline/N=1000-8       	       6	   1839868 ns/op	   28594 B/op	       7 allocs/op
BenchmarkPipeline/N=100000-8     	       1	  68848083 ns/op	   28832 B/op	      10 allocs/op
BenchmarkPipeline/N=1000000-8    	       1	 652115208 ns/op	   33912 B/op	      13 allocs/op
BenchmarkPipelineDedupChurn/N=10-8         	       9	   1243366 ns/op	 3546817 B/op	     303 allocs/op
BenchmarkPipelineDedupChurn/N=1000-8       	       2	   5012166 ns/op	 3683264 B/op	    3556 allocs/op
BenchmarkPipelineDedupChurn/N=100000-8     	       1	 189788958 ns/op	17523504 B/op	  372777 allocs/op
BenchmarkPipelineDedupChurn/N=1000000-8    	       1	1997392375 ns/op	164519200 B/op	 3749198 allocs/op
```

The dedup map is the bottleneck — it keeps a seen-set that grows with unique event count. Default capacity (10K) keeps memory reasonable; at 1M events the map resets once. Everything else is channels and goroutines and they're fast.
