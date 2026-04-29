package rrtcount

import (
	"math"
	"math/rand"

	"golang.org/x/exp/slices"
)

// randomizedResponseMatrix is the same edge-RR routine used by rrkcore;
// duplicated here to keep the algorithms self-contained without forcing
// rrkcore to expose private helpers.
func randomizedResponseMatrix(adj map[int][]int, n int, epsilon float64, seed int64) [][]int {
	prob := 1.0 / (math.Exp(epsilon) + 1.0)
	rng := rand.New(rand.NewSource(seed))
	m := make([][]int, n)
	for i := 0; i < n; i++ {
		m[i] = make([]int, n)
	}
	for i := 0; i < n; i++ {
		nbrs := adj[i]
		for j := i + 1; j < n; j++ {
			has := slices.Contains(nbrs, j)
			flipped := rng.Float64() < prob
			present := has
			if flipped {
				present = !has
			}
			if present {
				m[i][j] = 1
				m[j][i] = 1
			}
		}
	}
	return m
}
