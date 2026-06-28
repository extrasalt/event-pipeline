// Package pipeline provides a generic, concurrent streaming pipeline
// library with per-stage metadata, fan-out, and pluggable stages.
//
// A pipeline is a sequence of Stage[T] implementations chained via
// buffered channels. Each stage runs in its own goroutine. The caller
// feeds items into the input channel and drains the output channel;
// once the output is fully drained, per-stage metadata is available.
//
// Usage:
//
//	p := pipeline.Pipeline[int]{
//	    Stages: []pipeline.Stage[int]{
//	        pipeline.Map("double", func(x int) (int, error) { return x * 2, nil }),
//	    },
//	    Buffer: 64,
//	}
//	out, collectMeta := p.Run(ctx, in)
//	for v := range out { ... }
//	meta := collectMeta()
package pipeline

import "context"

// StageMeta holds per-stage counters collected after a pipeline run.
type StageMeta struct {
	Name      string
	Processed int64
	Errors    int64
	Dropped   int64
}

// Stage is the interface every pipeline stage must implement.
//
// Process reads from in, applies the stage's transform, and writes
// results to out. It must return when ctx is cancelled or in is
// closed and fully drained. Process is called in a single goroutine;
// for concurrent processing use FanOut.
//
// Meta returns a snapshot of the stage's counters. It is safe to
// call concurrently.
type Stage[T any] interface {
	Process(ctx context.Context, in <-chan T, out chan<- T)
	Meta() StageMeta
}

// Pipeline chains a sequence of stages into a single streaming
// pipeline. Set Stages to the ordered list of stages and Buffer to
// the channel buffer size for each inter-stage link (defaults to 64).
type Pipeline[T any] struct {
	Stages []Stage[T]
	Buffer int
}

// Run starts the pipeline. Each stage runs in its own goroutine,
// connected by buffered channels of size p.Buffer. The output channel
// is returned along with a metadata closure. The closure must be
// called after the output channel has been fully drained to obtain
// accurate per-stage counters.
func (p *Pipeline[T]) Run(ctx context.Context, in <-chan T) (<-chan T, func() []StageMeta) {
	out := in
	for _, s := range p.Stages {
		out = runStage(ctx, s, out, p.Buffer)
	}
	return out, func() []StageMeta {
		m := make([]StageMeta, len(p.Stages))
		for i, s := range p.Stages {
			m[i] = s.Meta()
		}
		return m
	}
}

// runStage spawns a goroutine that runs stage s in a loop over
// in, writing results to a new buffered channel of size buf.
func runStage[T any](ctx context.Context, s Stage[T], in <-chan T, buf int) <-chan T {
	if buf <= 0 {
		buf = 64
	}
	out := make(chan T, buf)
	go func() {
		defer close(out)
		s.Process(ctx, in, out)
	}()
	return out
}
