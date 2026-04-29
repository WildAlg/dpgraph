// Package rrkcore is the randomized-response baseline for k-core
// decomposition. Each user perturbs their adjacency row with the standard
// edge-RR mechanism (flip each potential edge bit with probability
// 1/(e^eps + 1)); the curator then runs an unbiased degree-rescaling and
// the classical heap-based peeling algorithm on the noisy graph.
//
// This baseline is included to put kcoreLDP / kcoreCDP results in
// context — it is generally less accurate but conceptually simpler.
package rrkcore

import (
	"container/heap"
	"math"

	"github.com/WildAlg/dpgraph/pkg/algo"
	"github.com/WildAlg/dpgraph/pkg/sim"
)

const Name = "rr-kcore"

func init() {
	algo.Register(Name, func(cfg sim.Config) algo.Algorithm { return &Algo{} })
}

// Algo implements algo.Algorithm.
type Algo struct {
	matrix [][]int
	cores  map[int]float64
}

// Name implements algo.Algorithm.
func (*Algo) Name() string { return Name }

// Setup applies edge-RR to produce a symmetric n*n 0/1 matrix.
func (a *Algo) Setup(ctx *sim.RunCtx) error {
	cfg := ctx.Cfg
	a.matrix = randomizedResponseMatrix(ctx.Graph.AdjacencyList, cfg.GraphSize, cfg.Epsilon, cfg.Seed)
	return nil
}

// Run performs the heap-based peeling on the noisy graph and records each
// vertex's removal-time degree as its core estimate.
func (a *Algo) Run(ctx *sim.RunCtx) error {
	n := len(a.matrix)
	scale := math.Exp(ctx.Cfg.Epsilon) + 1.0
	denom := math.Exp(ctx.Cfg.Epsilon) - 1.0

	degree := make([]float64, n)
	for u := 0; u < n; u++ {
		for j := 0; j < n; j++ {
			degree[u] += (float64(a.matrix[u][j])*scale - 1) / denom
		}
	}

	nodes := make([]*node, n)
	h := make(minHeap, 0, n)
	for u := 0; u < n; u++ {
		nd := &node{id: u, deg: degree[u]}
		nodes[u] = nd
		h = append(h, nd)
	}
	for i, nd := range h {
		nd.idx = i
	}
	heap.Init(&h)

	a.cores = make(map[int]float64, n)
	removed := make([]bool, n)
	for h.Len() > 0 {
		top := heap.Pop(&h).(*node)
		u := top.id
		if removed[u] {
			continue
		}
		a.cores[u] = top.deg
		removed[u] = true
		for v := 0; v < n; v++ {
			if a.matrix[u][v] == 1 && !removed[v] {
				degree[v]--
				h.update(nodes[v], degree[v])
			}
		}
	}

	// Account for one logical broadcast of each peeled vertex's core
	// number from the curator out (8 bytes per vertex).
	ctx.SimulateBroadcast(0, n*8)
	return nil
}

// Finalize emits per-vertex core numbers.
func (a *Algo) Finalize(ctx *sim.RunCtx, sink *sim.ResultSink) error {
	for v, d := range a.cores {
		if err := sink.WriteVertex(v, formatF(d)); err != nil {
			return err
		}
	}
	return nil
}
