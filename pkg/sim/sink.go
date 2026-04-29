package sim

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ResultSink writes per-vertex or scalar algorithm results in the legacy
// output format ("v: value" lines, optionally preceded by header lines that
// algorithms may want to emit). It also exposes WriteHeader for things like
// "Rounds: 8" that the original codebase prints at the top.
//
// The legacy filename schema produced by FilenameForConfig matches the
// pre-refactor experiments.Runner so scripts/get_results.py keeps working.
type ResultSink struct {
	w  io.WriteCloser
	bw *bufio.Writer
}

// NewResultSink opens the output file at path, creating any missing parent
// directories.
func NewResultSink(path string) (*ResultSink, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return &ResultSink{w: f, bw: bufio.NewWriter(f)}, nil
}

// WriteHeader writes a free-form header line.
func (s *ResultSink) WriteHeader(format string, args ...any) error {
	_, err := fmt.Fprintf(s.bw, format, args...)
	if err != nil {
		return err
	}
	if len(format) == 0 || format[len(format)-1] != '\n' {
		_, err = fmt.Fprintln(s.bw)
	}
	return err
}

// WriteVertex writes "v: value" for a per-vertex result.
func (s *ResultSink) WriteVertex(v int, value any) error {
	_, err := fmt.Fprintf(s.bw, "%d: %v\n", v, value)
	return err
}

// WriteScalar writes a single labelled scalar (e.g. "Triangles: 1234").
func (s *ResultSink) WriteScalar(label string, value any) error {
	_, err := fmt.Fprintf(s.bw, "%s: %v\n", label, value)
	return err
}

// Close flushes and closes the underlying file.
func (s *ResultSink) Close() error {
	if err := s.bw.Flush(); err != nil {
		_ = s.w.Close()
		return err
	}
	return s.w.Close()
}

// FilenameForConfig returns the legacy output filename for cfg, matching
// experiments.Runner's format string in the parent repo:
//
//	{graph}_{algo}_{factor:.2f}_{bias01}_{noise01}_{biasFactor}_{runID}_{numWorkers}_{epsilon:.2f}_{tag}.txt
func FilenameForConfig(cfg Config) string {
	b2i := func(b bool) int {
		if b {
			return 1
		}
		return 0
	}
	return fmt.Sprintf("%s_%s_%.2f_%d_%d_%d_%d_%d_%.2f_%s.txt",
		cfg.Graph, cfg.AlgoName, cfg.Factor, b2i(cfg.Bias), b2i(cfg.Noise),
		cfg.BiasFactor, cfg.RunID, cfg.NumWorkers, cfg.Epsilon, cfg.OutputTag)
}
