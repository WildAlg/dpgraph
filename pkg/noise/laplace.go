package noise

import (
	googlenoise "github.com/google/differential-privacy/go/v2/noise"
)

// Laplace returns a google-dp Laplace mechanism. Provided as a thin re-export
// so algorithms only need to import dpgraph/pkg/noise, not the upstream
// library directly.
func Laplace() googlenoise.Noise {
	return googlenoise.Laplace()
}
