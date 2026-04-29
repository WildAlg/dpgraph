// Package rrtcount is the randomized-response baseline for triangle
// counting. After edge-flip RR is applied (same as rrkcore), each entry of
// the rescaled adjacency matrix Y[i][j] is the unbiased estimator of the
// presence of edge (i,j); the triangle count estimate is the sum over all
// triples (i,j,k) of Y[i][j] * Y[j][k] * Y[i][k].
//
// The triple loop is the work; we shard the outermost index across workers
// so that wall-clock time scales with cfg.NumWorkers and the framework
// records realistic simulated network cost for the per-worker partial sums.
package rrtcount

import (
	"math"

	"github.com/WildAlg/dpgraph/pkg/algo"
	"github.com/WildAlg/dpgraph/pkg/sim"
)

const Name = "rr-tcount"

func init() {
	algo.Register(Name, func(cfg sim.Config) algo.Algorithm { return &Algo{} })
}

// Algo implements algo.Algorithm.
type Algo struct {
	matrix [][]int
	Y      [][]float64
	count  float64
}

// Name implements algo.Algorithm.
func (*Algo) Name() string { return Name }

// Setup applies edge-RR and pre-computes the unbiased Y matrix.
func (a *Algo) Setup(ctx *sim.RunCtx) error {
	cfg := ctx.Cfg
	a.matrix = randomizedResponseMatrix(ctx.Graph.AdjacencyList, cfg.GraphSize, cfg.Epsilon, cfg.Seed)
	scale := math.Exp(cfg.Epsilon) + 1.0
	denom := math.Exp(cfg.Epsilon) - 1.0
	n := cfg.GraphSize
	a.Y = make([][]float64, n)
	for i := 0; i < n; i++ {
		a.Y[i] = make([]float64, n)
		for j := 0; j < n; j++ {
			a.Y[i][j] = (float64(a.matrix[i][j])*scale - 1.0) / denom
		}
	}
	return nil
}

// Run computes the triangle-count estimate, sharding the i-loop across the
// configured number of workers.
func (a *Algo) Run(ctx *sim.RunCtx) error {
	n := len(a.Y)
	if n < 3 {
		return nil
	}
	results := ctx.RunRound(0, func(w *sim.WorkerCtx) any {
		// i ranges over [Offset, Offset+Workload) ∩ [0, n-2).
		lo := w.Offset
		hi := w.Offset + w.Workload
		if hi > n-2 {
			hi = n - 2
		}
		var local float64
		for i := lo; i < hi; i++ {
			for j := i + 1; j < n-1; j++ {
				yij := a.Y[i][j]
				if yij == 0 {
					continue
				}
				for k := j + 1; k < n; k++ {
					local += yij * a.Y[j][k] * a.Y[i][k]
				}
			}
		}
		w.SimulateSend(8) // a single float64
		return local
	})
	for _, r := range results {
		a.count += r.(float64)
	}
	return nil
}

// Finalize emits the scalar triangle-count estimate.
func (a *Algo) Finalize(ctx *sim.RunCtx, sink *sim.ResultSink) error {
	return sink.WriteScalar("Triangle Count Approx", a.count)
}
