// Package tcountcdp ports the central-DP triangle-counting algorithm onto
// the dpgraph framework. Phases:
//
//  1. A k-core CDP pass settles the LDS (eps/4 budget).
//  2. The curator applies edge-RR globally and publishes X (eps/4).
//  3. Curator computes the noisy max out-degree (eps/4).
//  4. Curator counts triangles through oriented outgoing edges with
//     Laplace noise (eps/4).
//
// Privacy budget is split evenly across the four sub-steps, matching the
// original triangle-counting-cdp.go.
package tcountcdp

import (
	"math"
	"math/rand"
	"sort"

	googlenoise "github.com/google/differential-privacy/go/v2/noise"
	"golang.org/x/exp/slices"

	"github.com/WildAlg/dpgraph/algos/kcorecdp"
	"github.com/WildAlg/dpgraph/pkg/algo"
	"github.com/WildAlg/dpgraph/pkg/lds"
	dpnoise "github.com/WildAlg/dpgraph/pkg/noise"
	"github.com/WildAlg/dpgraph/pkg/sim"
)

const Name = "triangle_countingCDP"

func init() {
	algo.Register(Name, func(cfg sim.Config) algo.Algorithm { return &Algo{} })
}

// Algo implements algo.Algorithm.
type Algo struct {
	kc          *kcorecdp.Algo
	X           [][]bool
	maxNoisyOut float64
	count       float64
}

// Name implements algo.Algorithm.
func (*Algo) Name() string { return Name }

// Setup runs the k-core CDP setup with eps/4.
func (a *Algo) Setup(ctx *sim.RunCtx) error {
	cfg := ctx.Cfg
	subCfg := cfg
	subCfg.Epsilon = cfg.Epsilon / 4
	subCtx := sim.NewRunCtx(subCfg, ctx.Graph)
	subCtx.Latency = ctx.Latency
	subCtx.Metrics = ctx.Metrics
	a.kc = &kcorecdp.Algo{}
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
	return nil
}

// Run executes the three remaining super-steps on the curator.
func (a *Algo) Run(ctx *sim.RunCtx) error {
	cfg := ctx.Cfg
	n := cfg.GraphSize
	epsQuarter := cfg.Epsilon / 4
	rng := rand.New(rand.NewSource(cfg.Seed + 1))

	// Stage 1: publish RR over the full graph.
	for id := 0; id < n; id++ {
		nbrs := ctx.Graph.AdjacencyList[id]
		var noised []int
		if cfg.Noise {
			noised = randomizedResponse(epsQuarter, nbrs, n, id, rng)
		} else {
			noised = append([]int(nil), nbrs...)
		}
		sort.Ints(noised)
		for _, j := range noised {
			if j < 0 || j >= n {
				continue
			}
			a.X[id][j] = true
			a.X[j][id] = true
		}
	}
	ctx.SimulateBroadcast(0, n*n/8) // ~bit per cell of the X matrix

	// Stage 2: max noisy out-degree.
	for id := 0; id < n; id++ {
		nbrs := ctx.Graph.AdjacencyList[id]
		out := orientOutEdges(id, nbrs, a.kc.LDS, rng)
		geom := dpnoise.NewGeom(epsQuarter)
		noisy := float64(len(out)) + float64(geom.TwoSidedGeometric())
		if noisy > a.maxNoisyOut {
			a.maxNoisyOut = noisy
		}
	}
	ctx.SimulateBroadcast(1, 8)

	// Stage 3: enumerate triangles via oriented outgoing edges.
	u := math.Exp(epsQuarter) + 1.0
	denom := math.Exp(epsQuarter) - 1.0
	for id := 0; id < n; id++ {
		nbrs := ctx.Graph.AdjacencyList[id]
		out := orientOutEdges(id, nbrs, a.kc.LDS, rng)
		sort.Ints(out)
		end := int(math.Min(a.maxNoisyOut, float64(len(out))))
		var local float64
		for j := 0; j < end; j++ {
			for k := j + 1; k < end; k++ {
				b := 0.0
				if a.X[out[j]][out[k]] {
					b = 1.0
				}
				local += (b*u - 1) / denom
			}
		}
		noisy, err := googlenoise.Laplace().AddNoiseFloat64(local, 1, a.maxNoisyOut, epsQuarter/2, 0)
		if err == nil {
			a.count += noisy
		} else {
			a.count += local
		}
	}
	return nil
}

// Finalize emits the triangle-count estimate.
func (a *Algo) Finalize(ctx *sim.RunCtx, sink *sim.ResultSink) error {
	return sink.WriteScalar("Triangle Count Approx", a.count)
}

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
