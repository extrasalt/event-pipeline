package pipeline

import "context"

// Map creates a MapStage that transforms each input item using fn.
func Map[T any](name string, fn func(T) (T, error)) *MapStage[T] {
	return &MapStage[T]{Core: Core{Name: name}, Fn: fn}
}

// MapStage transforms each item by applying Fn.
type MapStage[T any] struct {
	Core
	Fn func(T) (T, error)
}

func (s *MapStage[T]) Process(ctx context.Context, in <-chan T, out chan<- T) {
	runCore(&s.Core, ctx, in, out, func(v T) (T, bool, error) {
		r, err := s.Fn(v)
		return r, true, err
	})
}
