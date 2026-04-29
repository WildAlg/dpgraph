// Package kcorecore holds the noise-and-threshold logic shared between the
// kcoreLDP and kcoreCDP algorithms. It is internal to the algos tree so that
// reference implementations can deduplicate without exposing private types
// to library consumers.
package kcorecore

import (
	"math"

	"github.com/WildAlg/dpgraph/pkg/graph"
	"github.com/WildAlg/dpgraph/pkg/noise"
)

// Vertex is the per-vertex algorithm state for hierarchical k-core
// algorithms. ID is the global vertex index; Neighbours is the (unmodified)
// adjacency list inherited from the graph; RoundThreshold is computed once
// at setup from the noisy degree.
type Vertex struct {
	ID             int
	CurrentLevel   int
	NextLevel      int
	PermanentZero  int
	RoundThreshold int
	Neighbours     []int
}

// BuildShard converts an adjacency-list shard into a map[ID]*Vertex with
// computed round thresholds. addNoise / biasFactor follow the original
// kcore-ldp.go:loadGraphWorker semantics.
func BuildShard(shard *graph.Graph, lambda, levelsPerGroup float64, addNoise bool, biasFactor int) (map[int]*Vertex, int) {
	out := make(map[int]*Vertex, len(shard.AdjacencyList))
	maxRT := 0
	for nodeID, nbrs := range shard.AdjacencyList {
		degree := len(nbrs)
		noised := int64(degree)
		if addNoise {
			geom := noise.NewGeom(lambda / 2.0)
			noised += geom.TwoSidedGeometric()
			bias := float64(biasFactor) *
				(2.0 * math.Exp(lambda)) /
				(math.Exp(2*lambda) - 1)
			noised -= int64(math.Min(bias, float64(noised)))
			noised += 1
		}
		threshold := math.Ceil(LogToBase(int(noised), 2)) * levelsPerGroup
		rt := int(threshold) + 1
		out[nodeID] = &Vertex{
			ID:             nodeID,
			CurrentLevel:   0,
			PermanentZero:  1,
			RoundThreshold: rt,
			Neighbours:     nbrs,
		}
		if rt > maxRT {
			maxRT = rt
		}
	}
	return out, maxRT
}

// LogToBase computes log_b(a) using log2.
func LogToBase(a int, b float64) float64 {
	return math.Log2(float64(a)) / math.Log2(b)
}

// LevelsPerGroup returns the original kcoreLDP/kcoreCDP convention:
// ceil(log_{1+phi}(n)) / 4.
func LevelsPerGroup(n int, phi float64) float64 {
	return math.Ceil(LogToBase(n, 1.0+phi)) / 4
}

// RoundsParam returns ceil(4 * log_{1+phi}(n)^1.2), the round count used by
// both kcoreLDP and kcoreCDP.
func RoundsParam(n int, phi float64) float64 {
	return math.Ceil(4.0 * math.Pow(LogToBase(n, 1.0+phi), 1.2))
}
