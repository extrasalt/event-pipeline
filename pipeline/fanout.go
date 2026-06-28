package pipeline

import (
	"context"
	"sync"
)

// FanOut spawns n goroutines, each running stage s against the shared
// input channel in. Results from all workers are merged into a single
// output channel. Since all workers share the same Stage instance,
// metadata counters must use atomic operations (as Core does).
func FanOut[T any](ctx context.Context, in <-chan T, n int, s Stage[T]) <-chan T {
	out := make(chan T, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Process(ctx, in, out)
		}()
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
