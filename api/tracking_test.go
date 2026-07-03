package api

import (
	"context"
	"testing"

	"github.com/extrasalt/event-pipeline/pipeline"
)

func TestTrackingPipeline_ValidateRejectsEmptyType(t *testing.T) {
	ctx := context.Background()
	in := make(chan *TrackingEvent, 2)
	in <- &TrackingEvent{Type: "page_view"}
	in <- &TrackingEvent{Type: ""}
	close(in)

	out, collectMeta := trackingPipeline.Run(ctx, in)
	var results []*TrackingEvent
	for e := range out {
		results = append(results, e)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 to pass, got %d", len(results))
	}

	meta := collectMeta()
	var validateMeta pipeline.StageMeta
	for _, m := range meta {
		if m.Name == "validate" {
			validateMeta = m
			break
		}
	}
	if validateMeta.Processed != 1 {
		t.Fatalf("validate processed: expected 1, got %d", validateMeta.Processed)
	}
	if validateMeta.Errors != 1 {
		t.Fatalf("validate errors: expected 1, got %d", validateMeta.Errors)
	}
}

func TestTrackingPipeline_Dedup(t *testing.T) {
	ctx := context.Background()
	in := make(chan *TrackingEvent, 3)
	in <- &TrackingEvent{ID: "dup1", Type: "purchase"}
	in <- &TrackingEvent{ID: "dup1", Type: "purchase"}
	in <- &TrackingEvent{ID: "unique", Type: "page_view"}
	close(in)

	out, collectMeta := trackingPipeline.Run(ctx, in)
	var results []*TrackingEvent
	for e := range out {
		results = append(results, e)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 (1 deduped), got %d", len(results))
	}

	meta := collectMeta()
	for _, m := range meta {
		if m.Name == "dedup" {
			if m.Processed != 2 {
				t.Fatalf("dedup processed: expected 2, got %d", m.Processed)
			}
			if m.Dropped != 1 {
				t.Fatalf("dedup dropped: expected 1, got %d", m.Dropped)
			}
			break
		}
	}
}

func TestTrackingPipeline_NormalizeTruncation(t *testing.T) {
	ctx := context.Background()
	longAgent := ""
	for i := 0; i < 600; i++ {
		longAgent += "a"
	}
	ip := "1112.2222.3333.4444.5555.6666.7777.8888.9999.0000"

	in := make(chan *TrackingEvent, 2)
	in <- &TrackingEvent{ID: "1", Type: "purchase", UserAgent: longAgent, IP: ip}
	close(in)

	out, _ := trackingPipeline.Run(ctx, in)
	var results []*TrackingEvent
	for e := range out {
		results = append(results, e)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if len(results[0].UserAgent) > 500 {
		t.Fatalf("user agent should be truncated to 500, got %d", len(results[0].UserAgent))
	}
	if len(results[0].IP) > 45 {
		t.Fatalf("ip should be truncated to 45, got %d", len(results[0].IP))
	}
}

func TestTrackingPipeline_ChurnEnrich(t *testing.T) {
	ctx := context.Background()
	in := make(chan *TrackingEvent, 2)
	in <- &TrackingEvent{ID: "1", Type: "purchase"}
	in <- &TrackingEvent{ID: "2", Type: "page_view"}
	close(in)

	out, _ := trackingPipeline.Run(ctx, in)
	var results []*TrackingEvent
	for e := range out {
		results = append(results, e)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	for _, e := range results {
		if e.ChurnProb == 0 {
			t.Fatalf("event %s should have churn probability set", e.ID)
		}
	}
}

func TestTrackingPipeline_AllEventTypesPass(t *testing.T) {
	ctx := context.Background()
	types := []string{"page_view", "click", "add_to_cart", "checkout", "payment_info", "purchase", "lead"}
	in := make(chan *TrackingEvent, len(types))
	for _, typ := range types {
		in <- &TrackingEvent{ID: typ, Type: typ}
	}
	close(in)

	out, collectMeta := trackingPipeline.Run(ctx, in)
	var results []*TrackingEvent
	for e := range out {
		results = append(results, e)
	}

	if len(results) != 7 {
		t.Fatalf("expected all 7 types to pass, got %d", len(results))
	}

	meta := collectMeta()
	for _, m := range meta {
		if m.Name == "validate" && m.Errors > 0 {
			t.Fatalf("validate should not error on valid events: %+v", m)
		}
	}
}

func TestTrackingPipeline_MetadataPerRun(t *testing.T) {
	ctx := context.Background()

	in1 := make(chan *TrackingEvent, 2)
	in1 <- &TrackingEvent{ID: "a", Type: "purchase"}
	in1 <- &TrackingEvent{ID: "b", Type: "page_view"}
	close(in1)

	out1, collectMeta1 := trackingPipeline.Run(ctx, in1)
	for range out1 {
	}
	meta1 := collectMeta1()

	in2 := make(chan *TrackingEvent, 2)
	in2 <- &TrackingEvent{ID: "c", Type: "click"}
	in2 <- &TrackingEvent{ID: "d", Type: "lead"}
	close(in2)

	out2, collectMeta2 := trackingPipeline.Run(ctx, in2)
	for range out2 {
	}
	meta2 := collectMeta2()

	for i, m := range meta1 {
		if m.Processed+m.Errors+m.Dropped != 2 {
			t.Fatalf("run 1 stage %s: expected total 2, got processed=%d errors=%d dropped=%d",
				m.Name, m.Processed, m.Errors, m.Dropped)
		}
		total2 := meta2[i].Processed + meta2[i].Errors + meta2[i].Dropped
		if total2 != 2 {
			t.Fatalf("run 2 stage %s: expected total 2 (per-run delta), got processed=%d errors=%d dropped=%d",
				meta2[i].Name, meta2[i].Processed, meta2[i].Errors, meta2[i].Dropped)
		}
	}
}
