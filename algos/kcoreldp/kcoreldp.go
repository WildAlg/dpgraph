// Package kcoreldp ports the LDP k-core decomposition algorithm onto the
// dpgraph framework. It is the canonical example of a multi-round LDP graph
// algorithm: each super-step, a worker reads neighbour levels from the
// shared LDS, samples geometric noise on its noisy neighbour count, and
// reports a level-bump bit per local vertex back to the coordinator.
package kcoreldp

import (
	"math"

	"github.com/WildAlg/dpgraph/algos/internal/kcorecore"
	"github.com/WildAlg/dpgraph/pkg/algo"
	"github.com/WildAlg/dpgraph/pkg/lds"
	"github.com/WildAlg/dpgraph/pkg/noise"
	"github.com/WildAlg/dpgraph/pkg/sim"
)

// Name is the dispatch key. Kept as the legacy "kcoreLDP" identifier so
// existing YAML configs and result-filename schemas continue to work.
const Name = "kcoreLDP"

func init() {
	algo.Register(Name, func(cfg sim.Config) algo.Algorithm { return &Algo{} })
}

// Algo implements algo.Algorithm for k-core-LDP. Exported fields after
// Setup (LDS, LevelsPerGroup, NumRounds, MaxPublicRT) are read by other
// algorithms (e.g. tcountldp) that compose with it.
type Algo struct {
	LDS            *lds.LDS
	LevelsPerGroup float64
	NumRounds      int
	MaxPublicRT    int

	psi         float64
	roundsParam float64
	step1Lambda float64 // epsilon * factor
	step2Lambda float64 // epsilon * (1 - factor)
	addNoise    bool
	biasFactor  int

	// shards[w] maps GLOBAL vertex ID -> per-vertex algorithm state for
	// vertices that worker w owns.
	shards []map[int]*kcorecore.Vertex
}

// Name implements algo.Algorithm.
func (*Algo) Name() string { return Name }

// Setup loads the partitioned shards, samples one-shot noisy degrees per
// vertex, and computes per-vertex round thresholds.
func (a *Algo) Setup(ctx *sim.RunCtx) error {
	cfg := ctx.Cfg
	n := cfg.GraphSize
	a.psi = cfg.Phi
	a.addNoise = cfg.Noise
	a.biasFactor = cfg.BiasFactor
	a.LevelsPerGroup = kcorecore.LevelsPerGroup(n, cfg.Phi)
	a.roundsParam = kcorecore.RoundsParam(n, cfg.Phi)
	a.NumRounds = int(a.roundsParam)
	a.step1Lambda = cfg.Epsilon * cfg.Factor
	a.step2Lambda = cfg.Epsilon * (1.0 - cfg.Factor)

	a.LDS = lds.New(n, a.LevelsPerGroup)
	a.shards = make([]map[int]*kcorecore.Vertex, cfg.NumWorkers)

	maxRT := 0
	for w, shard := range ctx.Shards {
		shardMap, shardMax := kcorecore.BuildShard(shard, a.step1Lambda, a.LevelsPerGroup, a.addNoise, a.biasFactor)
		a.shards[w] = shardMap
		if shardMax > maxRT {
			maxRT = shardMax
		}
	}
	a.MaxPublicRT = maxRT
	return nil
}

// Run executes the LDS round loop, fanning out workers via ctx.RunRound.
func (a *Algo) Run(ctx *sim.RunCtx) error {
	rounds := a.NumRounds - 2
	if a.MaxPublicRT < rounds {
		rounds = a.MaxPublicRT
	}
	for round := 0; round < rounds; round++ {
		groupIdx := a.LDS.GroupForLevel(uint(round))
		results := ctx.RunRound(round, func(wctx *sim.WorkerCtx) any {
			return a.workerRound(wctx, round, float64(groupIdx))
		})
		for w, msg := range results {
			levels := msg.([]int)
			for localID, bump := range levels {
				if bump == 1 {
					_ = a.LDS.LevelIncrease(uint(workerOffset(ctx, w) + localID))
				}
			}
		}
	}
	return nil
}

func workerOffset(ctx *sim.RunCtx, workerID int) int {
	chunk := ctx.Cfg.GraphSize / ctx.Cfg.NumWorkers
	return workerID * chunk
}

func (a *Algo) workerRound(wctx *sim.WorkerCtx, round int, groupIdx float64) []int {
	nextLevels := make([]int, wctx.Workload)
	shard := a.shards[wctx.ID]
	for _, v := range shard {
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
			nextLevels[v.ID-wctx.Offset] = 1
		} else {
			v.PermanentZero = 0
		}
	}
	wctx.SimulateSend(len(nextLevels) * 4)
	return nextLevels
}

// Finalize emits per-vertex estimated core numbers in the legacy
// "%d: %.4f\n" format compatible with scripts/get_results.py.
func (a *Algo) Finalize(ctx *sim.RunCtx, sink *sim.ResultSink) error {
	cores := EstimateCoreNumbers(a.LDS, ctx.Cfg.GraphSize, a.psi, 0.5, a.LevelsPerGroup)
	for i, v := range cores {
		if err := sink.WriteVertex(i, formatF(v)); err != nil {
			return err
		}
	}
	return nil
}

// EstimateCoreNumbers produces the per-vertex core-number upper bound from a
// settled LDS. Exposed so that tcountldp / tcountcdp / external tools can
// reuse it without re-deriving the formula.
func EstimateCoreNumbers(l *lds.LDS, n int, phi, lambda, levelsPerGroup float64) []float64 {
	out := make([]float64, n)
	twoPlusLambda := 2.0 + lambda
	onePlusPhi := 1.0 + phi
	for i := 0; i < n; i++ {
		nodeLevel, err := l.GetLevel(uint(i))
		if err != nil {
			continue
		}
		fracNum := float64(nodeLevel) + 1.0
		power := math.Max(math.Floor(fracNum/levelsPerGroup)-1.0, 0.0)
		out[i] = twoPlusLambda * math.Pow(onePlusPhi, power)
	}
	return out
}
