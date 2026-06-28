package pipeline

import (
	"context"
	"sync/atomic"
)

// Core is an embedded helper that provides atomic counters and a
// shared Meta implementation for concrete stage types. Stages that
// process items one-by-one should embed Core and call runCore.
type Core struct {
	Name      string
	Processed atomic.Int64
	Errors    atomic.Int64
	Dropped   atomic.Int64
}

// Meta returns a StageMeta snapshot of the current counters.
func (c *Core) Meta() StageMeta {
	return StageMeta{Name: c.Name, Processed: c.Processed.Load(), Errors: c.Errors.Load(), Dropped: c.Dropped.Load()}
}

// runCore drives the standard read-process-send loop that most stages
// share. For each item read from in, fn is called. If fn returns an
// error the item is counted as an error and skipped. If fn returns
// keep=false the item is counted as dropped. Otherwise the result is
// sent to out and counted as processed. The loop exits on ctx
// cancellation or when in is closed.
func runCore[T any](c *Core, ctx context.Context, in <-chan T, out chan<- T, fn func(T) (T, bool, error)) {
	for {
		select {
		case <-ctx.Done():
			return
		case v, ok := <-in:
			if !ok {
				return
			}
			r, keep, err := fn(v)
			if err != nil {
				c.Errors.Add(1)
				continue
			}
			if !keep {
				c.Dropped.Add(1)
				continue
			}
			select {
			case out <- r:
				c.Processed.Add(1)
			case <-ctx.Done():
				return
			}
		}
	}
}
