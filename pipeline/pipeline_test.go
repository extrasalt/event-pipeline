package pipeline

import (
	"context"
	"fmt"
	"testing"
)

type testEvent struct {
	ID    string
	Type  string
	Value float64
	Score float64
}

func (e *testEvent) SetChurnScore(s ChurnScore) { e.Score = s.Probability }

func TestMap(t *testing.T) {
	ctx := context.Background()
	in := make(chan int, 5)
	for i := 0; i < 5; i++ {
		in <- i
	}
	close(in)

	p := Pipeline[int]{Stages: []Stage[int]{Map("double", func(x int) (int, error) { return x * 2, nil })}, Buffer: 4}
	out, meta := p.Run(ctx, in)

	var got []int
	for v := range out {
		got = append(got, v)
	}
	if len(got) != 5 || got[0] != 0 || got[4] != 8 {
		t.Fatalf("unexpected: %v", got)
	}
	m := meta()
	if m[0].Name != "double" || m[0].Processed != 5 {
		t.Fatalf("bad meta: %+v", m[0])
	}
}

func TestFilter(t *testing.T) {
	ctx := context.Background()
	in := make(chan int, 10)
	for i := 0; i < 10; i++ {
		in <- i
	}
	close(in)

	p := Pipeline[int]{
		Stages: []Stage[int]{Filter("even", func(x int) (bool, error) { return x%2 == 0, nil })},
		Buffer: 4,
	}
	out, meta := p.Run(ctx, in)

	var got []int
	for v := range out {
		got = append(got, v)
	}
	if len(got) != 5 {
		t.Fatalf("expected 5 evens, got %d", len(got))
	}
	m := meta()
	if m[0].Processed != 5 || m[0].Dropped != 5 {
		t.Fatalf("bad meta: %+v", m[0])
	}
}

func TestGenerate(t *testing.T) {
	ctx := context.Background()
	in := make(chan int)
	close(in)

	p := Pipeline[int]{Stages: []Stage[int]{Generate("gen", func(ctx context.Context, emit func(int)) {
		for i := 0; i < 3; i++ {
			emit(i)
		}
	})}, Buffer: 4}
	out, meta := p.Run(ctx, in)

	var got []int
	for v := range out {
		got = append(got, v)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3, got %d", len(got))
	}
	m := meta()
	if m[0].Processed != 3 {
		t.Fatalf("bad meta: %+v", m[0])
	}
}

func TestIf(t *testing.T) {
	ctx := context.Background()
	in := make(chan int, 4)
	for i := 1; i <= 4; i++ {
		in <- i
	}
	close(in)

	p := Pipeline[int]{Stages: []Stage[int]{
		If("oddDoubleEvenTriple",
			func(x int) bool { return x%2 != 0 },
			func(x int) (int, error) { return x * 2, nil },
			func(x int) (int, error) { return x * 3, nil },
		),
	}, Buffer: 4}
	out, _ := p.Run(ctx, in)

	var got []int
	for v := range out {
		got = append(got, v)
	}
	want := []int{2, 6, 6, 12}
	for i, v := range got {
		if v != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestReduce(t *testing.T) {
	ctx := context.Background()
	in := make(chan int, 5)
	for i := 1; i <= 5; i++ {
		in <- i
	}
	close(in)

	sum, m := Reduce(ctx, in, 0, func(a int, v int) (int, error) { return a + v, nil })
	if sum != 15 {
		t.Fatalf("expected 15, got %d", sum)
	}
	if m.Processed != 5 {
		t.Fatalf("bad meta: %+v", m)
	}
}

func TestCollect(t *testing.T) {
	ctx := context.Background()
	in := make(chan int, 7)
	for i := 0; i < 7; i++ {
		in <- i
	}
	close(in)

	out := Collect(ctx, in, 3)
	var batches [][]int
	for b := range out {
		batches = append(batches, b)
	}
	if len(batches) != 3 || len(batches[0]) != 3 || len(batches[2]) != 1 {
		t.Fatalf("unexpected batches: %v", len(batches))
	}
}

func TestDeduplicate(t *testing.T) {
	ctx := context.Background()
	in := make(chan *testEvent, 5)
	in <- &testEvent{ID: "a"}
	in <- &testEvent{ID: "b"}
	in <- &testEvent{ID: "a"}
	in <- &testEvent{ID: "c"}
	in <- &testEvent{ID: "b"}
	close(in)

	p := Pipeline[*testEvent]{Stages: []Stage[*testEvent]{
		Deduplicate("dedup", func(e *testEvent) string { return e.ID }, 100),
	}, Buffer: 4}
	out, meta := p.Run(ctx, in)

	var got []*testEvent
	for v := range out {
		got = append(got, v)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 unique, got %d", len(got))
	}
	m := meta()
	if m[0].Processed != 3 || m[0].Dropped != 2 {
		t.Fatalf("bad meta: %+v", m[0])
	}
}

func TestChurnEnrich(t *testing.T) {
	ctx := context.Background()
	in := make(chan *testEvent, 2)
	in <- &testEvent{ID: "1", Type: "purchase"}
	in <- &testEvent{ID: "2", Type: "purchase"}
	close(in)

	p := Pipeline[*testEvent]{Stages: []Stage[*testEvent]{
		ChurnEnrich[*testEvent]("enrich"),
	}, Buffer: 4}
	out, meta := p.Run(ctx, in)

	var got []*testEvent
	for v := range out {
		got = append(got, v)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
	for _, e := range got {
		if e.Score < 0 || e.Score >= 1 {
			t.Fatalf("bad score: %f", e.Score)
		}
	}
	m := meta()
	if m[0].Processed != 2 {
		t.Fatalf("bad meta: %+v", m[0])
	}
}

func TestPipelineMultiStage(t *testing.T) {
	ctx := context.Background()
	in := make(chan int, 10)
	for i := 0; i < 10; i++ {
		in <- i
	}
	close(in)

	p := Pipeline[int]{
		Stages: []Stage[int]{
			Map("double", func(x int) (int, error) { return x * 2, nil }),
			Filter("gt5", func(x int) (bool, error) { return x > 5, nil }),
		},
		Buffer: 8,
	}
	out, meta := p.Run(ctx, in)

	var got []int
	for v := range out {
		got = append(got, v)
	}
	if len(got) != 7 {
		t.Fatalf("expected 7, got %d: %v", len(got), got)
	}
	m := meta()
	if len(m) != 2 || m[0].Name != "double" || m[1].Name != "gt5" {
		t.Fatalf("bad metas: %+v", m)
	}
}

func TestReduceStage(t *testing.T) {
	ctx := context.Background()
	in := make(chan int, 5)
	for i := 1; i <= 5; i++ {
		in <- i
	}
	close(in)

	p := Pipeline[int]{
		Stages: []Stage[int]{ReduceStage("sum", func(a, b int) (int, error) { return a + b, nil })},
		Buffer: 4,
	}
	out, _ := p.Run(ctx, in)

	var got []int
	for v := range out {
		got = append(got, v)
	}
	if len(got) != 1 || got[0] != 15 {
		t.Fatalf("expected [15], got %v", got)
	}
}

func TestPipelineFilterReduce(t *testing.T) {
	ctx := context.Background()
	in := make(chan int, 10)
	for i := 1; i <= 10; i++ {
		in <- i
	}
	close(in)

	p := Pipeline[int]{
		Stages: []Stage[int]{
			Filter("even", func(x int) (bool, error) { return x%2 == 0, nil }),
			ReduceStage("sum", func(a, b int) (int, error) { return a + b, nil }),
		},
		Buffer: 8,
	}
	out, meta := p.Run(ctx, in)

	var got []int
	for v := range out {
		got = append(got, v)
	}
	if len(got) != 1 || got[0] != 30 {
		t.Fatalf("expected [30], got %v", got)
	}
	m := meta()
	if len(m) != 2 {
		t.Fatalf("expected 2 stage metas, got %d", len(m))
	}
	if m[0].Name != "even" || m[0].Processed != 5 || m[0].Dropped != 5 {
		t.Fatalf("bad filter meta: %+v", m[0])
	}
	if m[1].Name != "sum" || m[1].Processed != 1 {
		t.Fatalf("bad reduce meta: %+v", m[1])
	}
}

func TestStageMeta_LatencyAndThroughput(t *testing.T) {
	ctx := context.Background()
	in := make(chan int, 10)
	for i := 0; i < 10; i++ {
		in <- i
	}
	close(in)

	p := Pipeline[int]{
		Stages: []Stage[int]{Map("double", func(x int) (int, error) { return x * 2, nil })},
		Buffer: 4,
	}
	out, collectMeta := p.Run(ctx, in)
	for range out {
	}
	m := collectMeta()
	if m[0].LatencyNs <= 0 {
		t.Fatalf("expected positive latency, got %d", m[0].LatencyNs)
	}
	if m[0].Throughput <= 0 {
		t.Fatalf("expected positive throughput, got %f", m[0].Throughput)
	}
}

func TestFanOut(t *testing.T) {
	ctx := context.Background()
	in := make(chan int, 100)
	for i := 0; i < 100; i++ {
		in <- i
	}
	close(in)

	s := Map("double", func(x int) (int, error) { return x * 2, nil })
	out := FanOut(ctx, in, 4, s)

	count := 0
	for range out {
		count++
	}
	if count != 100 {
		t.Fatalf("expected 100, got %d", count)
	}
}

func TestCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	in := make(chan int, 5)
	for i := 0; i < 5; i++ {
		in <- i
	}
	close(in)

	p := Pipeline[int]{Stages: []Stage[int]{Map("id", func(x int) (int, error) { return x, nil })}, Buffer: 4}
	cancel()
	out, _ := p.Run(ctx, in)
	for range out {
	}
}

var sink int

func BenchmarkPipeline(b *testing.B) {
	for _, n := range []int{10, 1000, 100000, 1000000} {
		b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			ctx := context.Background()
			p := Pipeline[int]{
				Stages: []Stage[int]{
					Map("id", func(x int) (int, error) { return x, nil }),
					Filter("even", func(x int) (bool, error) { return x%2 == 0, nil }),
				},
				Buffer: 1024,
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				in := make(chan int, 1024)
				go func() {
					for j := 0; j < n; j++ {
						in <- j
					}
					close(in)
				}()
				out, _ := p.Run(ctx, in)
				for range out {
					sink++
				}
			}
		})
	}
}

func BenchmarkPipelineDedupChurn(b *testing.B) {
	for _, n := range []int{10, 1000, 100000, 1000000} {
		b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			ctx := context.Background()
			p := Pipeline[*testEvent]{
				Stages: []Stage[*testEvent]{
					Map("validate", func(e *testEvent) (*testEvent, error) {
						if e.ID == "" {
							return e, fmt.Errorf("empty id")
						}
						return e, nil
					}),
					Filter("purchaseOnly", func(e *testEvent) (bool, error) { return e.Type == "purchase", nil }),
					Deduplicate("dedup", func(e *testEvent) string { return e.ID }, 100000),
					ChurnEnrich[*testEvent]("enrich"),
				},
				Buffer: 1024,
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				in := make(chan *testEvent, 1024)
				go func() {
					for j := 0; j < n; j++ {
						typ := "purchase"
						if j%3 == 0 {
							typ = "click"
						}
						in <- &testEvent{ID: fmt.Sprintf("evt-%d", j), Type: typ}
					}
					close(in)
				}()
				out, _ := p.Run(ctx, in)
				for range out {
					sink++
				}
			}
		})
	}
}
