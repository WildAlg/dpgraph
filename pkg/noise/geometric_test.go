package noise

import (
	"math"
	"testing"
)

// TestTwoSidedGeometricDistribution verifies that the empirical mean is near
// zero and the variance is in the right ballpark for a two-sided geometric
// at lambda=1. We do not exercise determinism here because the upstream
// google-dp rand package draws from a process-global crypto source.
func TestTwoSidedGeometricDistribution(t *testing.T) {
	g := NewGeom(1.0)
	const N = 2000
	var sum, sumSq float64
	for i := 0; i < N; i++ {
		v := float64(g.TwoSidedGeometric())
		sum += v
		sumSq += v * v
	}
	mean := sum / N
	variance := sumSq/N - mean*mean
	if math.Abs(mean) > 0.5 {
		t.Errorf("mean=%v, want |mean|<0.5", mean)
	}
	if variance < 0.1 || variance > 5.0 {
		t.Errorf("variance=%v, want roughly in [0.1, 5]", variance)
	}
}
