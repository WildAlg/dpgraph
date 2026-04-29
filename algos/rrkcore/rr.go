package rrkcore

import (
	"math"
	"math/rand"
	"strconv"

	"golang.org/x/exp/slices"
)

// randomizedResponseMatrix builds the symmetric n*n 0/1 adjacency matrix
// after applying edge-flip RR with probability 1/(e^eps + 1) to the upper
// triangular part of the original adjacency.
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

func formatF(v float64) string { return strconv.FormatFloat(v, 'f', 4, 64) }
