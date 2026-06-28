package pipeline

import "context"

// Collect batches items from in into slices of batchSize.
func Collect[T any](ctx context.Context, in <-chan T, batchSize int) <-chan []T {
	out := make(chan []T)
	go func() {
		defer close(out)
		batch := make([]T, 0, batchSize)
		flush := func() {
			if len(batch) > 0 {
				out <- batch
			}
		}
		defer flush()
		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-in:
				if !ok {
					return
				}
				batch = append(batch, v)
				if len(batch) >= batchSize {
					select {
					case out <- batch:
					case <-ctx.Done():
						return
					}
					batch = make([]T, 0, batchSize)
				}
			}
		}
	}()
	return out
}
