package pipeline

import "context"

// Deduplicate creates a DeduplicateStage that drops duplicate keys.
func Deduplicate[T any](name string, keyFn func(T) string, cap int) *DeduplicateStage[T] {
	if cap <= 0 {
		cap = 10000
	}
	return &DeduplicateStage[T]{Core: Core{Name: name}, KeyFn: keyFn, Cap: cap}
}

// DeduplicateStage drops items whose key has already been seen.
type DeduplicateStage[T any] struct {
	Core
	KeyFn func(T) string
	Cap   int
}

func (s *DeduplicateStage[T]) Process(ctx context.Context, in <-chan T, out chan<- T) {
	seen := make(map[string]struct{}, s.Cap)
	runCore(&s.Core, ctx, in, out, func(v T) (T, bool, error) {
		key := s.KeyFn(v)
		if _, exists := seen[key]; exists {
			return v, false, nil
		}
		seen[key] = struct{}{}
		if len(seen) >= s.Cap {
			seen = make(map[string]struct{}, s.Cap)
		}
		return v, true, nil
	})
}
