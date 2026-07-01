package pipeline

import "context"

// Reduce folds items from in into an accumulator using fn.
func Reduce[T, R any](ctx context.Context, in <-chan T, init R, fn func(R, T) (R, error)) (R, StageMeta) {
	result := init
	var m StageMeta
	for {
		select {
		case <-ctx.Done():
			return result, m
		case v, ok := <-in:
			if !ok {
				return result, m
			}
			var err error
			result, err = fn(result, v)
			if err != nil {
				m.Errors++
			} else {
				m.Processed++
			}
		}
	}
}
