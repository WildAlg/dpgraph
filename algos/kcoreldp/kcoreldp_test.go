package kcoreldp

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/WildAlg/dpgraph/pkg/algo"
	"github.com/WildAlg/dpgraph/pkg/sim"
)

// repoTestdata returns testdata/toy_10node_adj relative to the dpgraph
// module root, regardless of which package's working directory the test
// runs in.
func repoTestdata(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	// algos/kcoreldp/<this file> → ../../testdata/toy_10node_adj
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "testdata", "toy_10node_adj")
}

// TestKcoreLDPEndToEnd runs the algorithm with noise disabled (deterministic
// path) and checks that the 4-clique gets the highest core estimate, the
// triangle the middle, and the path the lowest.
func TestKcoreLDPEndToEnd(t *testing.T) {
	tmp := t.TempDir()
	cfg := sim.Config{
		Graph:      "toy",
		GraphPath:  repoTestdata(t),
		GraphSize:  10,
		AlgoName:   Name,
		NumWorkers: 2,
		Epsilon:    1.0,
		Phi:        0.5,
		Factor:     0.8,
		Noise:      false,
		BiasFactor: 1,
		Seed:       1,
		OutputDir:  tmp,
		OutputTag:  "test",
	}
	if err := algo.Run(cfg); err != nil {
		t.Fatal(err)
	}
	body, err := readResult(tmp, cfg)
	if err != nil {
		t.Fatal(err)
	}
	cores := parseCores(t, body)
	if len(cores) != 10 {
		t.Fatalf("got %d core entries, want 10", len(cores))
	}

	// 4-clique vertices (0..3) should dominate the triangle (4..6) and the
	// path tail (7..9). LDP estimates are upper bounds, so we compare
	// strictly greater-or-equal between groups.
	cliqueMin := minOf(cores[:4])
	triMax := maxOf(cores[4:7])
	pathMax := maxOf(cores[7:])
	if cliqueMin < triMax {
		t.Errorf("4-clique min %.4f < triangle max %.4f", cliqueMin, triMax)
	}
	if triMax < pathMax {
		t.Errorf("triangle max %.4f < path max %.4f", triMax, pathMax)
	}
}

func readResult(dir string, cfg sim.Config) (string, error) {
	path := filepath.Join(dir, sim.FilenameForConfig(cfg))
	b, err := readFile(path)
	return string(b), err
}

func parseCores(t *testing.T, body string) []float64 {
	t.Helper()
	out := make([]float64, 0, 10)
	for _, line := range strings.Split(strings.TrimSpace(body), "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		var v float64
		if _, err := fmtSscanf(strings.TrimSpace(parts[1]), &v); err != nil {
			t.Fatalf("parse %q: %v", line, err)
		}
		out = append(out, v)
	}
	return out
}

func minOf(xs []float64) float64 {
	m := xs[0]
	for _, x := range xs[1:] {
		if x < m {
			m = x
		}
	}
	return m
}

func maxOf(xs []float64) float64 {
	m := xs[0]
	for _, x := range xs[1:] {
		if x > m {
			m = x
		}
	}
	return m
}
