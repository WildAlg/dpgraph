// Package tcountldp ports the LDP triangle-counting algorithm onto the
// dpgraph framework. It runs in four super-steps:
//
//  1. A k-core LDP pass settles the LDS so each vertex knows its level.
//  2. workerRR: each worker applies edge-RR to its shard and the
//     coordinator publishes the noisy adjacency X.
//  3. workerMaxOutDegree: each worker reports the noisy max out-degree of
//     its shard; the coordinator publishes the global max.
//  4. workerCountTriangles: each worker enumerates triangles through
//     outgoing edges, applies Laplace noise, and reports a local count.
//
// Total privacy budget is split as eps/4 for each of the four sub-steps
// (k-core, RR, max-out-degree, count).
package tcountldp

import (
	"math"
	"math/rand"
	"sort"

	googlenoise "github.com/google/differential-privacy/go/v2/noise"
	"golang.org/x/exp/slices"

	"github.com/WildAlg/dpgraph/algos/kcoreldp"
	"github.com/WildAlg/dpgraph/pkg/algo"
	"github.com/WildAlg/dpgraph/pkg/lds"
	dpnoise "github.com/WildAlg/dpgraph/pkg/noise"
	"github.com/WildAlg/dpgraph/pkg/sim"
)

const Name = "triangle_countingLDP"

func init() {
	algo.Register(Name, func(cfg sim.Config) algo.Algorithm { return &Algo{} })
}

// Algo implements algo.Algorithm.
type Algo struct {
	kc          *kcoreldp.Algo
	X           [][]bool  // noisy adjacency, n x n
	shards      [][][]int // per-worker raw adjacency-as-list-of-int, indexed by local id
	maxNoisyOut float64
	count       float64
}

// Name implements algo.Algorithm.
func (*Algo) Name() string { return Name }

// Setup runs the k-core LDP setup with eps/4, allocates X, and stores the
// per-worker shards as []int slices indexed by local vertex id.
func (a *Algo) Setup(ctx *sim.RunCtx) error {
	cfg := ctx.Cfg
	// Sub-algorithm gets a copy of the config with eps/4.
	subCfg := cfg
	subCfg.Epsilon = cfg.Epsilon / 4
	subCtx := sim.NewRunCtx(subCfg, ctx.Graph)
	subCtx.Latency = ctx.Latency
	subCtx.Metrics = ctx.Metrics

	a.kc = &kcoreldp.Algo{}
	if err := a.kc.Setup(subCtx); err != nil {
		return err
	}
	if err := a.kc.Run(subCtx); err != nil {
		return err
	}

	a.X = make([][]bool, cfg.GraphSize)
	for i := range a.X {
		a.X[i] = make([]bool, cfg.GraphSize)
	}

	a.shards = make([][][]int, cfg.NumWorkers)
	chunk := cfg.GraphSize / cfg.NumWorkers
	for w, shard := range ctx.Shards {
		offset := w * chunk
		size := len(shard.AdjacencyList)
		shardList := make([][]int, size)
		for vid, nbrs := range shard.AdjacencyList {
			shardList[vid-offset] = nbrs
		}
		a.shards[w] = shardList
	}
	return nil
}

// Run executes the three triangle-counting super-steps.
func (a *Algo) Run(ctx *sim.RunCtx) error {
	cfg := ctx.Cfg
	n := cfg.GraphSize
	epsQuarter := cfg.Epsilon / 4

	// Stage 1: workerRR. Each worker applies RR to its shard and returns
	// the noisy neighbour lists. The coordinator commits these into X.
	rrOut := ctx.RunRound(0, func(w *sim.WorkerCtx) any {
		out := make([][]int, len(a.shards[w.ID]))
		for localID, nbrs := range a.shards[w.ID] {
			globalID := w.Offset + localID
			if cfg.Noise {
				out[localID] = randomizedResponse(epsQuarter, nbrs, n, globalID, w.RNG())
			} else {
				cp := append([]int(nil), nbrs...)
				out[localID] = cp
			}
			sort.Ints(out[localID])
		}
		// Approx message size: total ints sent * 4 bytes.
		bytes := 0
		for _, s := range out {
			bytes += len(s) * 4
		}
		w.SimulateSend(bytes)
		return out
	})
	for w, msg := range rrOut {
		offset := w * (n / cfg.NumWorkers)
		for localID, nbrs := range msg.([][]int) {
			i := offset + localID
			for _, j := range nbrs {
				if j < 0 || j >= n {
					continue
				}
				a.X[i][j] = true
				a.X[j][i] = true
			}
		}
	}

	// Stage 2: workerMaxOutDegree. The framework's LatencyModel applies
	// to each worker's single-float reply automatically.
	maxOut := ctx.RunRound(1, func(w *sim.WorkerCtx) any {
		var workerMax float64
		for localID, nbrs := range a.shards[w.ID] {
			globalID := w.Offset + localID
			outEdges := orientOutEdges(globalID, nbrs, a.kc.LDS, w.RNG())
			geom := dpnoise.NewGeom(epsQuarter)
			noisy := float64(len(outEdges)) + float64(geom.TwoSidedGeometric())
			if noisy > workerMax {
				workerMax = noisy
			}
		}
		w.SimulateSend(8) // one float64
		return workerMax
	})
	for _, v := range maxOut {
		if d := v.(float64); d > a.maxNoisyOut {
			a.maxNoisyOut = d
		}
	}

	// Stage 3: workerCountTriangles. Enumerate triangles through oriented
	// outgoing edges, apply Laplace noise, and aggregate.
	tcOut := ctx.RunRound(2, func(w *sim.WorkerCtx) any {
		u := math.Exp(epsQuarter) + 1.0
		denom := math.Exp(epsQuarter) - 1.0
		var workerTC float64
		for localID, nbrs := range a.shards[w.ID] {
			globalID := w.Offset + localID
			outEdges := orientOutEdges(globalID, nbrs, a.kc.LDS, w.RNG())
			sort.Ints(outEdges)
			end := int(math.Min(a.maxNoisyOut, float64(len(outEdges))))
			var localTC float64
			for j := 0; j < end; j++ {
				for k := j + 1; k < end; k++ {
					b := 0.0
					if a.X[outEdges[j]][outEdges[k]] {
						b = 1.0
					}
					localTC += (b*u - 1) / denom
				}
			}
			noisy, err := googlenoise.Laplace().AddNoiseFloat64(localTC, 1, a.maxNoisyOut, epsQuarter/2, 0)
			if err == nil {
				workerTC += noisy
			} else {
				workerTC += localTC
			}
		}
		w.SimulateSend(8)
		return workerTC
	})
	for _, v := range tcOut {
		a.count += v.(float64)
	}
	return nil
}

// Finalize emits the scalar triangle count.
func (a *Algo) Finalize(ctx *sim.RunCtx, sink *sim.ResultSink) error {
	return sink.WriteScalar("Triangle Count Approx", a.count)
}

// orientOutEdges returns the subset of `nbrs` that are higher in the LDS
// than nodeID (with ties broken by a coin flip drawn from rng), i.e. the
// "outgoing" edges in the implicit DAG induced by the level ordering.
func orientOutEdges(nodeID int, nbrs []int, l *lds.LDS, rng *rand.Rand) []int {
	myLevel, _ := l.GetLevel(uint(nodeID))
	out := make([]int, 0, len(nbrs))
	for _, ngh := range nbrs {
		ngLevel, err := l.GetLevel(uint(ngh))
		if err != nil {
			continue
		}
		switch {
		case ngLevel > myLevel:
			out = append(out, ngh)
		case ngLevel == myLevel:
			if rng.Float64() <= 0.5 {
				out = append(out, ngh)
			}
		}
	}
	return out
}

// randomizedResponse mirrors the original triangle-counting-ldp.go flip
// logic: for each potential edge (nodeID, j) with j > nodeID, flip the bit
// independently with probability 1/(e^eps+1).
func randomizedResponse(epsilon float64, nbrs []int, n, nodeID int, rng *rand.Rand) []int {
	prob := 1.0 / (math.Exp(epsilon) + 1.0)
	out := make([]int, 0, len(nbrs))
	for j := nodeID + 1; j < n; j++ {
		has := slices.Contains(nbrs, j)
		flipped := rng.Float64() < prob
		present := has
		if flipped {
			present = !has
		}
		if present {
			out = append(out, j)
		}
	}
	return out
}
