package pipeline

import (
	"context"
	"math/rand"
)

// ChurnScore represents a simulated churn probability.
type ChurnScore struct {
	Probability float64
	Reason      string
}

// ChurnEnrichable is the constraint for ChurnEnrichStage.
type ChurnEnrichable interface {
	SetChurnScore(ChurnScore)
}

// ChurnEnrich creates a ChurnEnrichStage that assigns a simulated churn score.
func ChurnEnrich[T ChurnEnrichable](name string) *ChurnEnrichStage[T] {
	return &ChurnEnrichStage[T]{Core: Core{Name: name}}
}

// ChurnEnrichStage assigns each item a simulated churn score.
type ChurnEnrichStage[T ChurnEnrichable] struct {
	Core
}

func (s *ChurnEnrichStage[T]) Process(ctx context.Context, in <-chan T, out chan<- T) {
	runCore(&s.Core, ctx, in, out, func(v T) (T, bool, error) {
		v.SetChurnScore(ChurnScore{Probability: rand.Float64(), Reason: "simulated"})
		return v, true, nil
	})
}
