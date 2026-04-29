package sim

import (
	"math/rand"
	"sync"
	"time"

	"github.com/WildAlg/dpgraph/pkg/graph"
)

// Config is the framework-level run config. It mirrors the YAML schema used
// by the parent paper repo so existing config files continue to work, plus a
// few framework-only knobs (Seed, Latency, OutputDir).
//
// Algorithms read fields they care about; the framework does not validate
// fields outside its own concerns (it is up to the algorithm to error if,
// say, Phi is required but zero).
type Config struct {
	Graph         string
	GraphPath     string
	GraphSize     int
	AlgoName      string
	NumWorkers    int
	Epsilon       float64
	Phi           float64
	Factor        float64
	Bias          bool
	BiasFactor    int
	Noise         bool
	Runs          int
	RunID         int
	Seed          int64
	Bidirectional bool
	OutputDir     string
	OutputTag     string
	Latency       LatencyModel
}

// RunCtx is the run-scoped context handed to an Algorithm. It owns the
// graph, shards, latency model, metrics, shared state, and per-worker RNGs.
// Construct one via NewRunCtx, then call algo.Setup / algo.Run / algo.Finalize.
type RunCtx struct {
	Cfg     Config
	Graph   *graph.Graph
	Shards  []*graph.Graph
	Latency LatencyModel
	Metrics *Metrics
	State   *State

	rngs []*rand.Rand
}

// NewRunCtx loads the graph, partitions it, and prepares per-worker RNGs.
// If cfg.Latency is nil the latency model defaults to Zero{}.
func NewRunCtx(cfg Config, g *graph.Graph) *RunCtx {
	if cfg.Latency == nil {
		cfg.Latency = Zero{}
	}
	if cfg.NumWorkers <= 0 {
		cfg.NumWorkers = 1
	}
	rngs := make([]*rand.Rand, cfg.NumWorkers)
	for i := range rngs {
		rngs[i] = rand.New(rand.NewSource(cfg.Seed + int64(i)))
	}
	return &RunCtx{
		Cfg:     cfg,
		Graph:   g,
		Shards:  graph.Partition(g, cfg.GraphSize, cfg.NumWorkers),
		Latency: cfg.Latency,
		Metrics: NewMetrics(),
		State:   NewState(),
		rngs:    rngs,
	}
}

// SimulateBroadcast models a single coordinator-side message of `bytes`
// bytes (e.g. a state broadcast). Unlike WorkerCtx.SimulateSend it is meant
// to be called from the algorithm's main goroutine and counts as one
// message in the metrics, not numWorkers messages.
func (r *RunCtx) SimulateBroadcast(round, bytes int) {
	d := r.Latency.Delay(-1, -1, bytes, round)
	r.Metrics.recordSend(bytes, d)
	if d > 0 {
		time.Sleep(d)
	}
}

// WorkerCtx is what each Worker function receives. It exposes the worker's
// shard, an RNG, the shared state, and SimulateSend / SimulateRecv hooks
// that apply the run's LatencyModel.
type WorkerCtx struct {
	ID         int
	NumWorkers int
	Round      int
	Offset     int // first vertex ID owned by this worker
	Workload   int // number of vertices owned (= len(Shard.AdjacencyList))
	Shard      *graph.Graph
	State      *State
	rng        *rand.Rand
	latency    LatencyModel
	metrics    *Metrics
}

// RNG returns the worker's deterministic RNG (seeded from Cfg.Seed + workerID).
func (w *WorkerCtx) RNG() *rand.Rand { return w.rng }

// SimulateSend models a single message of the given byte size travelling
// from this worker to the coordinator. It records the delay in metrics and
// sleeps the calling goroutine for that delay.
//
// Sleeps from concurrent workers overlap in wall-clock time, which matches
// real distributed systems where parallel sends share no critical path.
func (w *WorkerCtx) SimulateSend(bytes int) {
	d := w.latency.Delay(w.ID, -1, bytes, w.Round)
	w.metrics.recordSend(bytes, d)
	if d > 0 {
		time.Sleep(d)
	}
}

// SimulateRecv models a coordinator-broadcast (or peer) message of the
// given byte size arriving at this worker. Same behaviour as SimulateSend
// but with src = -1.
func (w *WorkerCtx) SimulateRecv(bytes int) {
	d := w.latency.Delay(-1, w.ID, bytes, w.Round)
	w.metrics.recordSend(bytes, d)
	if d > 0 {
		time.Sleep(d)
	}
}

// WorkerFn is the per-shard computation. It may call SimulateSend /
// SimulateRecv as appropriate, and returns a single message of any type
// representing its round output. The framework collects these into a slice
// indexed by worker ID and returns them from RunRound.
type WorkerFn func(wctx *WorkerCtx) any

// RunRound launches one goroutine per worker, runs fn concurrently, joins,
// and returns the slice of return values indexed by worker ID. Round counter
// in WorkerCtx is set to the supplied round.
//
// If fn panics in a worker, the panic is propagated after the join so that
// the run aborts cleanly rather than leaking the goroutine.
func (r *RunCtx) RunRound(round int, fn WorkerFn) []any {
	out := make([]any, r.Cfg.NumWorkers)
	chunk := r.Cfg.GraphSize / r.Cfg.NumWorkers
	extra := r.Cfg.GraphSize % r.Cfg.NumWorkers
	var wg sync.WaitGroup
	wg.Add(r.Cfg.NumWorkers)
	r.Metrics.Rounds = round + 1
	for i := 0; i < r.Cfg.NumWorkers; i++ {
		offset := i * chunk
		workload := chunk
		if i == r.Cfg.NumWorkers-1 {
			workload = chunk + extra
		}
		wctx := &WorkerCtx{
			ID:         i,
			NumWorkers: r.Cfg.NumWorkers,
			Round:      round,
			Offset:     offset,
			Workload:   workload,
			Shard:      r.Shards[i],
			State:      r.State,
			rng:        r.rngs[i],
			latency:    r.Latency,
			metrics:    r.Metrics,
		}
		go func(idx int, wctx *WorkerCtx) {
			defer wg.Done()
			out[idx] = fn(wctx)
		}(i, wctx)
	}
	wg.Wait()
	return out
}
