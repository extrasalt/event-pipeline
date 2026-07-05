package pipeline

import (
	"context"
	"time"
)

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

// ReduceStage creates a ReduceStage that folds all input items into a
// single output using fn. Unlike the standalone Reduce function, it
// implements Stage[T] so it can be chained in Pipeline.Stages.
func ReduceStage[T any](name string, fn func(T, T) (T, error)) *reduceStage[T] {
	return &reduceStage[T]{Core: Core{Name: name}, Fn: fn}
}

// reduceStage folds items using Fn and emits the single result.
type reduceStage[T any] struct {
	Core
	Fn func(T, T) (T, error)
}

func (s *reduceStage[T]) Process(ctx context.Context, in <-chan T, out chan<- T) {
	start := time.Now()
	defer func() {
		s.LatencyNs.Add(time.Since(start).Nanoseconds())
	}()

	var acc T
	first := true

	for {
		select {
		case <-ctx.Done():
			return
		case v, ok := <-in:
			if !ok {
				if !first {
					select {
					case out <- acc:
						s.Processed.Add(1)
					case <-ctx.Done():
					}
				}
				return
			}
			if first {
				acc = v
				first = false
			} else {
				var err error
				acc, err = s.Fn(acc, v)
				if err != nil {
					s.Errors.Add(1)
				}
			}
		}
	}
}
