package sim

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/WildAlg/dpgraph/pkg/graph"
)

func makeCtx(t *testing.T, latency LatencyModel) *RunCtx {
	t.Helper()
	g := &graph.Graph{
		NumVertices: 4,
		AdjacencyList: map[int][]int{
			0: {1}, 1: {0}, 2: {3}, 3: {2},
		},
	}
	return NewRunCtx(Config{
		GraphSize:  4,
		NumWorkers: 2,
		Seed:       1,
		Latency:    latency,
	}, g)
}

func TestRunRoundFanout(t *testing.T) {
	ctx := makeCtx(t, Zero{})
	var calls int32
	out := ctx.RunRound(0, func(wctx *WorkerCtx) any {
		atomic.AddInt32(&calls, 1)
		if wctx.NumWorkers != 2 {
			t.Errorf("worker %d: NumWorkers=%d, want 2", wctx.ID, wctx.NumWorkers)
		}
		return wctx.ID * 10
	})
	if calls != 2 {
		t.Errorf("expected 2 worker invocations, got %d", calls)
	}
	if out[0].(int) != 0 || out[1].(int) != 10 {
		t.Errorf("unexpected return values: %v", out)
	}
}

func TestRunRoundSimulatedSendOverlap(t *testing.T) {
	// Two workers each "send" a 1ms message. Wall time should be ~1ms,
	// not 2ms — concurrent sleeps overlap.
	latency := BandwidthRTT{LinkSpeedBitsPerSec: 1e9, BaseRTT: 5 * time.Millisecond}
	ctx := makeCtx(t, latency)
	start := time.Now()
	_ = ctx.RunRound(0, func(wctx *WorkerCtx) any {
		wctx.SimulateSend(64)
		return nil
	})
	elapsed := time.Since(start)
	if elapsed > 12*time.Millisecond {
		t.Errorf("RunRound should overlap sleeps; got %v (expected ~5ms)", elapsed)
	}
	if elapsed < 4*time.Millisecond {
		t.Errorf("expected at least one ~5ms sleep, got %v", elapsed)
	}
	if ctx.Metrics.TotalMessages != 2 {
		t.Errorf("TotalMessages=%d, want 2", ctx.Metrics.TotalMessages)
	}
	if ctx.Metrics.TotalBytes != 128 {
		t.Errorf("TotalBytes=%d, want 128", ctx.Metrics.TotalBytes)
	}
}

func TestRunRoundDeterministicRNG(t *testing.T) {
	ctx1 := makeCtx(t, Zero{})
	ctx2 := makeCtx(t, Zero{})
	out1 := ctx1.RunRound(0, func(w *WorkerCtx) any { return w.RNG().Int63() })
	out2 := ctx2.RunRound(0, func(w *WorkerCtx) any { return w.RNG().Int63() })
	for i := range out1 {
		if out1[i] != out2[i] {
			t.Errorf("worker %d: deterministic seed mismatch (%v vs %v)", i, out1[i], out2[i])
		}
	}
}
