package pipeline

import "context"

// Generate creates a GenerateStage that produces items via emit.
func Generate[T any](name string, fn func(ctx context.Context, emit func(T))) *GenerateStage[T] {
	return &GenerateStage[T]{Core: Core{Name: name}, Fn: fn}
}

// GenerateStage produces items by calling Fn with an emit callback.
type GenerateStage[T any] struct {
	Core
	Fn func(ctx context.Context, emit func(T))
}

func (s *GenerateStage[T]) Process(ctx context.Context, in <-chan T, out chan<- T) {
	s.Fn(ctx, func(v T) {
		select {
		case out <- v:
			s.Processed.Add(1)
		case <-ctx.Done():
		}
	})
}
