package kcoreldp

import "strconv"

// formatF renders v as a fixed-4-decimal string, matching the "%.4f" lines
// the parent repo emits.
func formatF(v float64) string {
	return strconv.FormatFloat(v, 'f', 4, 64)
}
