package algo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/WildAlg/dpgraph/pkg/graph"
	"github.com/WildAlg/dpgraph/pkg/sim"
)

// Run is the framework's standard end-to-end driver. It loads the graph
// from cfg.GraphPath, looks up the algorithm by cfg.AlgoName, runs
// Setup/Run/Finalize, and writes results + metrics to cfg.OutputDir.
//
// Algorithm-package authors generally do not call Run directly — the CLI
// (cmd/dpgraph) and example programs do. They wire it up so that a YAML
// config plus a registered algorithm is enough to produce the same
// .txt output the parent repo's scripts/get_results.py expects, alongside
// a sibling .metrics.json.
func Run(cfg sim.Config) error {
	factory, ok := Get(cfg.AlgoName)
	if !ok {
		return fmt.Errorf("algo: no algorithm registered as %q", cfg.AlgoName)
	}
	a := factory(cfg)

	g, err := graph.Load(cfg.GraphPath, cfg.Bidirectional)
	if err != nil {
		return fmt.Errorf("algo: load graph: %w", err)
	}
	// Honour the explicit GraphSize from config if set; otherwise infer.
	if cfg.GraphSize == 0 {
		cfg.GraphSize = g.NumVertices
	}

	ctx := sim.NewRunCtx(cfg, g)

	endSetup := ctx.Metrics.PhaseTimer("setup")
	if err := a.Setup(ctx); err != nil {
		endSetup()
		return fmt.Errorf("algo %s: setup: %w", a.Name(), err)
	}
	endSetup()

	endAlgo := ctx.Metrics.PhaseTimer("algo")
	if err := a.Run(ctx); err != nil {
		endAlgo()
		return fmt.Errorf("algo %s: run: %w", a.Name(), err)
	}
	endAlgo()

	outPath := filepath.Join(cfg.OutputDir, sim.FilenameForConfig(cfg))
	sink, err := sim.NewResultSink(outPath)
	if err != nil {
		return fmt.Errorf("algo %s: open sink: %w", a.Name(), err)
	}
	if err := a.Finalize(ctx, sink); err != nil {
		_ = sink.Close()
		return fmt.Errorf("algo %s: finalize: %w", a.Name(), err)
	}
	if err := sink.Close(); err != nil {
		return err
	}
	ctx.Metrics.Finish()

	metricsPath := outPath + ".metrics.json"
	mf, err := os.Create(metricsPath)
	if err != nil {
		return fmt.Errorf("algo %s: open metrics: %w", a.Name(), err)
	}
	defer mf.Close()
	return ctx.Metrics.WriteJSON(mf)
}
