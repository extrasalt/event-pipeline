package pipeline

import "context"

// Filter creates a FilterStage that keeps items for which pred returns true.
func Filter[T any](name string, pred func(T) (bool, error)) *FilterStage[T] {
	return &FilterStage[T]{Core: Core{Name: name}, Pred: pred}
}

// FilterStage keeps items where Pred returns true; all others are dropped.
type FilterStage[T any] struct {
	Core
	Pred func(T) (bool, error)
}

func (s *FilterStage[T]) Process(ctx context.Context, in <-chan T, out chan<- T) {
	runCore(&s.Core, ctx, in, out, func(v T) (T, bool, error) {
		ok, err := s.Pred(v)
		return v, ok, err
	})
}
