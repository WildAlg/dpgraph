# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`dpgraph` is a Go framework for prototyping and benchmarking differentially private distributed graph algorithms. Module path: `github.com/WildAlg/dpgraph`. Ships reference LDP/CDP k-core and triangle-counting implementations plus randomized-response baselines.

## Common commands

```bash
# Build / test / lint (CI runs all of these — they must stay clean)
go build ./...
go test -race -count=1 ./...
go vet ./...
gofmt -l .                       # CI fails if this prints anything
golangci-lint run                # version pinned to v1.62.2 in CI

# Single package / single test
go test ./algos/kcoreldp/...
go test -run TestKcoreLDPDeterministic ./algos/kcoreldp

# Run the bundled example (smoke test against testdata/)
go run ./examples/hello_kcore

# CLI (YAML config compatible with the parent paper repo)
go run ./cmd/dpgraph -config_file path/to.yaml -workers 8 -link_mbps 1000 -rtt_ms 0.5
```

CI runs Go 1.25 (matching `go.mod`'s `go 1.25.6` directive).

## Architecture

The framework owns coordination, latency injection, RNG seeding, and metrics. Algorithm authors only implement a small plugin and never write goroutines, channels, locks, or `time.Sleep` themselves.

### Plugin contract (`pkg/algo`)

Every algorithm implements `algo.Algorithm` — `Name() / Setup(ctx) / Run(ctx) / Finalize(ctx, sink)` — and registers a `Factory` from `init()` via `algo.Register`. The CLI (`cmd/dpgraph/main.go`) blank-imports each algorithm package so registration happens at startup; **a new algorithm is not dispatchable until that blank import is added**.

`algo.Run(cfg sim.Config)` is the standard end-to-end driver: it loads the graph, looks up the algorithm by name, runs Setup → Run → Finalize, writes results to the legacy filename schema (`sim.FilenameForConfig`) plus a sibling `.metrics.json`.

### Run context (`pkg/sim`)

- `sim.RunCtx` is the run-scoped object handed to an Algorithm. It owns the loaded `*graph.Graph`, its per-worker `Shards`, the `LatencyModel`, `Metrics`, shared `State`, and per-worker RNGs seeded from `Cfg.Seed + workerID`.
- `RunCtx.RunRound(round, fn WorkerFn) []any` is **the** way to fan out a super-step: it spawns one goroutine per worker, runs `fn` concurrently with each worker's `WorkerCtx`, joins, and returns the per-worker outputs indexed by worker ID. Do not roll your own goroutines/`sync.WaitGroup`.
- `WorkerCtx.SimulateSend(bytes)` / `SimulateRecv(bytes)` apply the configured `LatencyModel`, record bytes/messages in metrics, and `time.Sleep` the calling goroutine. Concurrent worker sleeps overlap, so a round's wall time approximates `max(worker_compute + send_delay)` — matching real distributed systems where parallel sends share no critical path. Coordinator-side broadcasts go through `RunCtx.SimulateBroadcast` (counts as one message, not N).
- `sim.State` is a mutex-wrapped `map[string]any` for per-round broadcast data (e.g. an LDS, a noisy threshold) shared between coordinator and workers. Writes do **not** trigger latency injection — model that cost explicitly with `SimulateSend` / `SimulateRecv`.
- Latency models: `sim.Zero{}` (default; used by tests) and `sim.BandwidthRTT{LinkSpeedBitsPerSec, BaseRTT}`. The CLI only switches off `Zero` if `-link_mbps` or `-rtt_ms` is non-zero.

### Other packages

- `pkg/graph` — adjacency-list `Graph`, edge-list `Load`, vertex-range `Partition`. `AdjacencyList` is a map (vertex IDs may be sparse); `NumVertices` is the dense-range upper bound.
- `pkg/lds` — Levels Data Structure used by hierarchical k-core algorithms.
- `pkg/noise` — `Geom` (two-sided geometric) and a re-export of Google's Laplace sampler. All randomness in algorithms must flow through `WorkerCtx.RNG()` or `pkg/noise` so seeded runs are deterministic.
- `algos/internal/kcorecore` — shared per-vertex state used by both `kcoreldp` and `tcountldp`; example of how composed algorithms expose post-`Setup` fields (`LDS`, `LevelsPerGroup`, …) for downstream consumers.

### Adding an algorithm — non-obvious requirements

These are enforced by review (see `CONTRIBUTING.md`) and easy to miss:

1. New package goes under `algos/<name>/` and registers via `init()` in that package.
2. **Blank-import the new package in `cmd/dpgraph/main.go`** or the CLI cannot dispatch it.
3. Determinism: with `Cfg.Noise = false` and a fixed `Cfg.Seed`, output must be byte-identical across platforms. Tests must check this against `testdata/toy_10node_adj`.
4. Anywhere a message would cross a worker/coordinator boundary, call `SimulateSend` / `SimulateRecv` with a realistic byte size — the metrics block in the `.metrics.json` is meaningless otherwise.
5. Package doc comment must state the threat model (LDP / CDP / RR), what `Cfg.Epsilon` and `Cfg.Phi` control, and cite the paper.
6. The legacy result-filename schema (`sim.FilenameForConfig`) is an external contract used by the parent paper repo's `scripts/get_results.py` — do not change it.

## Conventions

- Idiomatic Go; gofmt enforced. Errors are returned, not logged-and-swallowed.
- One-line doc comments by default; algorithm packages take a longer threat-model paragraph.
- Don't add config knobs (`sim.Config` fields, YAML keys, CLI flags) that aren't exercised by a test — speculative flags rot fast.
