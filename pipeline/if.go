package pipeline

import "context"

// If creates an IfStage that branches on cond.
func If[T any](name string, cond func(T) bool, onTrue, onFalse func(T) (T, error)) *IfStage[T] {
	return &IfStage[T]{Core: Core{Name: name}, Cond: cond, OnTrue: onTrue, OnFalse: onFalse}
}

// IfStage branches on Cond: items satisfying Cond are transformed by OnTrue, others by OnFalse.
type IfStage[T any] struct {
	Core
	Cond    func(T) bool
	OnTrue  func(T) (T, error)
	OnFalse func(T) (T, error)
}

func (s *IfStage[T]) Process(ctx context.Context, in <-chan T, out chan<- T) {
	runCore(&s.Core, ctx, in, out, func(v T) (T, bool, error) {
		if s.Cond(v) {
			r, err := s.OnTrue(v)
			return r, true, err
		}
		r, err := s.OnFalse(v)
		return r, true, err
	})
}
