// Package kcorecdp ports the central-DP k-core decomposition algorithm
// onto the dpgraph framework. Unlike kcoreLDP, every vertex's noisy
// neighbour count is computed by a single trusted aggregator that sees the
// full graph; correspondingly the algorithm runs as one logical worker.
//
// Threat model: the curator is trusted and observes the entire adjacency
// list. epsilon-DP guarantees apply to released artifacts (the LDS levels
// and resulting core numbers), not to inputs.
package kcorecdp

import (
	"math"

	"github.com/WildAlg/dpgraph/algos/internal/kcorecore"
	"github.com/WildAlg/dpgraph/algos/kcoreldp"
	"github.com/WildAlg/dpgraph/pkg/algo"
	"github.com/WildAlg/dpgraph/pkg/lds"
	"github.com/WildAlg/dpgraph/pkg/noise"
	"github.com/WildAlg/dpgraph/pkg/sim"
)

// Name is the dispatch key.
const Name = "kcoreCDP"

func init() {
	algo.Register(Name, func(cfg sim.Config) algo.Algorithm { return &Algo{} })
}

// Algo implements algo.Algorithm for the central-DP k-core decomposition.
// It is structured similarly to kcoreldp.Algo but consolidates all vertices
// into a single shard and runs the round loop on the coordinator
// goroutine — there is no parallelism advantage in the central setting.
type Algo struct {
	LDS            *lds.LDS
	LevelsPerGroup float64
	NumRounds      int
	MaxPublicRT    int

	psi         float64
	step2Lambda float64
	addNoise    bool
	vertices    map[int]*kcorecore.Vertex
}

// Name implements algo.Algorithm.
func (*Algo) Name() string { return Name }

// Setup loads the (full) graph as a single shard and computes per-vertex
// round thresholds.
func (a *Algo) Setup(ctx *sim.RunCtx) error {
	cfg := ctx.Cfg
	n := cfg.GraphSize
	a.psi = cfg.Phi
	a.addNoise = cfg.Noise
	a.LevelsPerGroup = kcorecore.LevelsPerGroup(n, cfg.Phi)
	a.NumRounds = int(kcorecore.RoundsParam(n, cfg.Phi))
	step1Lambda := cfg.Epsilon * cfg.Factor
	a.step2Lambda = cfg.Epsilon * (1.0 - cfg.Factor)

	a.LDS = lds.New(n, a.LevelsPerGroup)
	a.vertices, a.MaxPublicRT = kcorecore.BuildShard(ctx.Graph, step1Lambda, a.LevelsPerGroup, a.addNoise, cfg.BiasFactor)
	return nil
}

// Run iterates the k-core round loop sequentially. We still call
// ctx.RunRound with a single virtual worker so the framework can record a
// "send" event and apply the configured LatencyModel to the broadcast
// of updated LDS levels each round.
func (a *Algo) Run(ctx *sim.RunCtx) error {
	rounds := a.NumRounds - 2
	if a.MaxPublicRT < rounds {
		rounds = a.MaxPublicRT
	}
	for round := 0; round < rounds; round++ {
		groupIdx := float64(a.LDS.GroupForLevel(uint(round)))
		// Two-pass per the original kcore-cdp: first decide nextLevel for
		// every vertex, then commit increases.
		toBump := make([]int, 0)
		for _, v := range a.vertices {
			if v.RoundThreshold == round {
				v.PermanentZero = 0
			}
			level, _ := a.LDS.GetLevel(uint(v.ID))
			v.CurrentLevel = int(level)
			if v.CurrentLevel != round || v.PermanentZero == 0 {
				continue
			}
			count := 0
			for _, ngh := range v.Neighbours {
				ngLevel, err := a.LDS.GetLevel(uint(ngh))
				if err != nil {
					continue
				}
				if int(ngLevel) == round {
					count++
				}
			}
			noisedCount := int64(count)
			if a.addNoise {
				scale := a.step2Lambda / (2.0 * float64(v.RoundThreshold))
				geom := noise.NewGeom(scale)
				noisedCount += geom.TwoSidedGeometric()
				extraBias := int64(3 * (2 * math.Exp(scale)) /
					math.Pow(math.Exp(2*scale)-1, 3))
				noisedCount += extraBias
			}
			threshold := int64(math.Pow(1.0+a.psi, groupIdx))
			if noisedCount > threshold {
				v.NextLevel = 1
				toBump = append(toBump, v.ID)
			} else {
				v.PermanentZero = 0
			}
		}
		// Model the cost of broadcasting per-vertex level updates from
		// the curator out to any consumers, parameterised by the size
		// of toBump (4 bytes per vertex bumped).
		ctx.SimulateBroadcast(round, len(toBump)*4)
		for _, id := range toBump {
			v := a.vertices[id]
			if v.NextLevel == 1 && v.PermanentZero != 0 {
				_ = a.LDS.LevelIncrease(uint(id))
			}
			v.NextLevel = 0
		}
	}
	return nil
}

// Finalize emits per-vertex estimated core numbers, matching the kcoreLDP
// output schema.
func (a *Algo) Finalize(ctx *sim.RunCtx, sink *sim.ResultSink) error {
	cores := kcoreldp.EstimateCoreNumbers(a.LDS, ctx.Cfg.GraphSize, a.psi, 0.5, a.LevelsPerGroup)
	for i, v := range cores {
		if err := sink.WriteVertex(i, formatF(v)); err != nil {
			return err
		}
	}
	return nil
}
