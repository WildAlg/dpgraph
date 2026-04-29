# dpgraph

[![CI](https://github.com/WildAlg/dpgraph/actions/workflows/ci.yml/badge.svg)](https://github.com/WildAlg/dpgraph/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/WildAlg/dpgraph.svg)](https://pkg.go.dev/github.com/WildAlg/dpgraph)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

`dpgraph` is a Go framework for **prototyping and benchmarking differentially private distributed graph algorithms**. You write a small `Algorithm` plugin; the framework handles graph loading, partitioning, worker fan-out, RNG seeding, metrics, and **simulated network latency** so your wall-clock numbers reflect a realistic distributed deployment instead of a single-machine goroutine soup.

It ships with reference implementations of the algorithms from *Practical and Accurate Local Differentially Private Graph Algorithms* (k-core decomposition and triangle counting under LDP and CDP, plus randomized-response baselines) so you can use them as templates or compare against them.

## Install

```bash
go get github.com/WildAlg/dpgraph
```

Requires Go 1.21+.

## 60-second quickstart

Run the bundled example against a 10-vertex toy graph:

```bash
git clone https://github.com/WildAlg/dpgraph
cd dpgraph
go run ./examples/hello_kcore
```

You should see per-vertex k-core estimates plus a metrics block showing simulated network time, message counts, and bytes transferred.

## Writing your own algorithm

Implement [`algo.Algorithm`](pkg/algo/algorithm.go) and register your factory in `init()`:

```go
package mydeg

import (
    "github.com/WildAlg/dpgraph/pkg/algo"
    "github.com/WildAlg/dpgraph/pkg/noise"
    "github.com/WildAlg/dpgraph/pkg/sim"
)

const Name = "private_degree"

func init() { algo.Register(Name, func(cfg sim.Config) algo.Algorithm { return &Algo{} }) }

type Algo struct{ noisyDeg []int64 }

func (*Algo) Name() string { return Name }

func (a *Algo) Setup(ctx *sim.RunCtx) error {
    a.noisyDeg = make([]int64, ctx.Cfg.GraphSize)
    return nil
}

func (a *Algo) Run(ctx *sim.RunCtx) error {
    out := ctx.RunRound(0, func(w *sim.WorkerCtx) any {
        local := make(map[int]int64)
        for v, nbrs := range w.Shard.AdjacencyList {
            g := noise.NewGeom(ctx.Cfg.Epsilon)
            local[v] = int64(len(nbrs)) + g.TwoSidedGeometric()
        }
        w.SimulateSend(len(local) * 16) // 16B per (vertex_id, count) pair
        return local
    })
    for _, msg := range out {
        for v, d := range msg.(map[int]int64) {
            a.noisyDeg[v] = d
        }
    }
    return nil
}

func (a *Algo) Finalize(ctx *sim.RunCtx, sink *sim.ResultSink) error {
    for i, d := range a.noisyDeg {
        if err := sink.WriteVertex(i, d); err != nil { return err }
    }
    return nil
}
```

That's it — no manual goroutines, channels, locks, or `time.Sleep` calls. The framework runs your `WorkerFn` in parallel, applies the configured `LatencyModel` whenever you call `SimulateSend` / `SimulateRecv`, and joins.

## What the framework gives you

| Concern | Provided by |
|---|---|
| Edge-list graph loader (whitespace `u v` format) | `pkg/graph.Load` |
| Vertex-range partitioning across workers | `pkg/graph.Partition` |
| Levels Data Structure for hierarchical k-core | `pkg/lds` |
| Two-sided geometric noise sampler | `pkg/noise.Geom` |
| Google Laplace noise (re-export) | `pkg/noise.Laplace` |
| Generic coordinator/worker round driver | `sim.RunCtx.RunRound` |
| Pluggable latency models | `sim.LatencyModel` (`Zero`, `BandwidthRTT`, or your own) |
| Per-worker deterministic RNG | `WorkerCtx.RNG()` |
| Shared run-scoped state | `sim.State` |
| Phase timers, message counters, bytes, simulated network ns | `sim.Metrics` |
| Output filename schema compatible with the original repo's analysis scripts | `sim.FilenameForConfig` |

## Latency models

```go
sim.Zero{}                                    // no delay (unit tests)
sim.BandwidthRTT{LinkSpeedBitsPerSec: 25e6,
                 BaseRTT: 1*time.Millisecond}  // 25 Mbps + 1ms RTT
```

Or implement your own:

```go
type LatencyModel interface {
    Delay(src, dst, bytes, round int) time.Duration
}
```

Concurrent worker sleeps overlap, so a round's wall time is roughly `max(worker_compute + send_delay)`, matching real distributed systems where parallel sends share no critical path.

## CLI

The `dpgraph` binary runs an algorithm against a YAML config matching the schema used by the parent paper repo:

```bash
go install github.com/WildAlg/dpgraph/cmd/dpgraph@latest
dpgraph -config_file mygraph-kcoreLDP.yaml -workers 8 -link_mbps 1000 -rtt_ms 0.5
```

Output files use the legacy filename schema so existing analysis scripts (`get_results.py` etc.) keep working; a sibling `<file>.metrics.json` is written alongside each result.

## Reference algorithms

| Name | Package | Type |
|---|---|---|
| `kcoreLDP` | [`algos/kcoreldp`](algos/kcoreldp) | Local-DP k-core |
| `kcoreCDP` | [`algos/kcorecdp`](algos/kcorecdp) | Central-DP k-core |
| `triangle_countingLDP` | [`algos/tcountldp`](algos/tcountldp) | Local-DP triangle count |
| `triangle_countingCDP` | [`algos/tcountcdp`](algos/tcountcdp) | Central-DP triangle count |
| `rr-kcore` | [`algos/rrkcore`](algos/rrkcore) | Randomized-response baseline |
| `rr-tcount` | [`algos/rrtcount`](algos/rrtcount) | Randomized-response baseline |

## Citing & paper

If you use `dpgraph` in research, please cite the originating paper. The parent repository contains the full PDF and Zenodo DOI: https://doi.org/10.5281/zenodo.15741879.

## Contributing

We welcome contributions of new algorithms, latency models, and benchmarks. See [CONTRIBUTING.md](CONTRIBUTING.md) for the bar new algorithms need to clear before they ship in `algos/`. Bug reports and discussion happen on GitHub Issues.

## License

MIT — see [LICENSE](LICENSE).
