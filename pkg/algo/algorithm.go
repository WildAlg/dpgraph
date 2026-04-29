// Package algo defines the plugin contract for dpgraph algorithms and a
// registry that lets the CLI dispatch by name.
//
// Implementing an algorithm is three methods: Setup (one-shot work on the
// loaded graph), Run (the round loop, calling RunCtx.RunRound to fan out
// workers), and Finalize (write per-vertex or scalar results). The
// framework owns coordination, latency injection, RNG seeding, and metrics.
package algo

import (
	"sync"

	"github.com/WildAlg/dpgraph/pkg/sim"
)

// Algorithm is the plugin contract. All three methods receive the same
// *sim.RunCtx; algorithms communicate between phases by writing to ctx.State
// or by storing per-instance fields on the implementing struct.
type Algorithm interface {
	// Name returns the dispatch key (e.g. "kcoreLDP"). Used for the
	// registry and for the legacy output-filename schema.
	Name() string

	// Setup runs once before the round loop. Use it to allocate per-vertex
	// data, sample one-shot noise, or compute static thresholds. The graph
	// and partitioned shards are already on ctx.
	Setup(ctx *sim.RunCtx) error

	// Run executes the algorithm body. Inside, call ctx.RunRound(round, fn)
	// for each super-step; the framework handles fan-out, latency, and
	// joining. Return when the algorithm has terminated.
	Run(ctx *sim.RunCtx) error

	// Finalize writes per-vertex or scalar results to sink. The framework
	// closes the sink after this returns.
	Finalize(ctx *sim.RunCtx, sink *sim.ResultSink) error
}

// Factory constructs a fresh Algorithm instance for a given Config. Each
// run gets its own instance; Factories should be cheap and stateless.
type Factory func(cfg sim.Config) Algorithm

var (
	regMu    sync.RWMutex
	registry = make(map[string]Factory)
)

// Register adds a Factory to the global registry. Calling Register twice
// with the same name overwrites the earlier entry; the convention is for
// each algorithm package to call Register from its init() so that simply
// importing the package makes the algorithm dispatchable.
func Register(name string, f Factory) {
	regMu.Lock()
	registry[name] = f
	regMu.Unlock()
}

// Get returns the Factory for the given name and a presence flag.
func Get(name string) (Factory, bool) {
	regMu.RLock()
	f, ok := registry[name]
	regMu.RUnlock()
	return f, ok
}

// Names returns the set of registered algorithm names. Order is not
// guaranteed.
func Names() []string {
	regMu.RLock()
	defer regMu.RUnlock()
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	return out
}
