// Package noise exposes the discrete-geometric (and, via google-dp, Laplace)
// noise primitives used by dpgraph algorithms. The two-sided geometric
// distribution is the standard discrete mechanism for adding integer noise
// to count queries under epsilon-DP.
//
// This implementation mirrors the binary-search sampler from
// github.com/google/differential-privacy/tree/main/go/v2/noise.
package noise

import (
	"math"

	"github.com/google/differential-privacy/go/v2/rand"
)

// Geom samples from a two-sided geometric distribution with parameter
// p = 1 - e^-lambda.
type Geom struct {
	lambda float64
}

// NewGeom constructs a sampler with parameter lambda; lambda must be > 2^-59
// for the truncation probability to remain below 1e-6 (see google-dp note).
func NewGeom(lambda float64) *Geom {
	return &Geom{lambda: lambda}
}

// geometric draws a sample from a one-sided geometric distribution truncated
// to int64 range.
func (g *Geom) geometric() int64 {
	if rand.Uniform() > -1.0*math.Expm1(-1.0*g.lambda*math.MaxInt64) {
		return math.MaxInt64
	}
	var left int64 = 0
	var right int64 = math.MaxInt64
	for left+1 < right {
		mid := left - int64(math.Floor((math.Log(0.5)+math.Log1p(math.Exp(g.lambda*float64(left-right))))/g.lambda))
		if mid <= left {
			mid = left + 1
		} else if mid >= right {
			mid = right - 1
		}
		q := math.Expm1(g.lambda*float64(left-mid)) / math.Expm1(g.lambda*float64(left-right))
		if rand.Uniform() <= q {
			right = mid
		} else {
			left = mid
		}
	}
	return right
}

// TwoSidedGeometric returns a sample from the symmetric two-sided geometric
// distribution centred at 0.
func (g *Geom) TwoSidedGeometric() int64 {
	var sample int64 = 0
	var sign int64 = -1
	for sample == 0 && sign == -1 {
		sample = g.geometric() - 1
		sign = int64(rand.Sign())
	}
	return sample * sign
}
