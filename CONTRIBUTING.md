# Contributing to dpgraph

Thanks for considering a contribution. `dpgraph` is community-curated: if your algorithm clears the bar below, we'll happily merge it into `algos/` so other researchers can run and compare against it.

## Adding a new algorithm

1. **Create a package under `algos/<name>/`.** Implement the [`algo.Algorithm`](pkg/algo/algorithm.go) interface (`Name`, `Setup`, `Run`, `Finalize`). Register your factory in an `init()` so importing the package is enough to make the algorithm dispatchable by name.

2. **Use the framework primitives.** Don't write your own goroutines, channel maps, `sync.WaitGroup`s, or `time.Sleep` calls — that's what `sim.RunCtx.RunRound` and the `LatencyModel` are for. Anywhere your algorithm sends a message between worker and coordinator, call `WorkerCtx.SimulateSend(bytes)` or `SimulateRecv(bytes)` so the run reflects realistic network cost.

3. **Be deterministic when seeded.** All randomness should flow through `WorkerCtx.RNG()` (which is seeded per-worker from `Cfg.Seed`) or through `pkg/noise` samplers. Tests must pass on every platform with a fixed seed.

4. **Wire your algorithm into `cmd/dpgraph/main.go`.** Add a blank import (`_ "github.com/WildAlg/dpgraph/algos/<name>"`) so the CLI registers it.

5. **Tests are required.** At minimum, an end-to-end test against `testdata/toy_10node_adj` verifying that the deterministic (noise-off) output is sensible. Algorithms that produce per-vertex outputs should compare ordering or relative magnitudes; scalar outputs should hit an exact expected value with `Cfg.Noise = false`.

6. **Document the threat model.** Add a one-paragraph package doc comment explaining what privacy guarantee the algorithm provides (local DP / central DP / RR baseline), what `Cfg.Epsilon` and `Cfg.Phi` control, and any references to the paper(s) that introduced it.

## PR checklist

- [ ] `go build ./...` clean
- [ ] `go test ./...` passes locally
- [ ] `go vet ./...` and `golangci-lint run` clean (CI enforces both)
- [ ] At least one end-to-end test against `testdata/`
- [ ] Algorithm registered via `init()` and blank-imported in `cmd/dpgraph/main.go`
- [ ] Package doc comment with threat model + paper reference
- [ ] README's "Reference algorithms" table updated
- [ ] If new public APIs are added, godoc comments on every exported symbol

## Style

- Idiomatic Go. We run `gofmt` (CI fails otherwise).
- Errors returned, not logged-and-swallowed.
- One-line doc comments are usually enough; algorithm-level threat-model paragraphs are the exception.
- Avoid adding configuration knobs that aren't exercised by your tests — speculative flags rot fast.

## Reporting bugs / proposing changes

Open an issue describing:
- What you ran (config + flags).
- What you expected.
- What you got.
- A minimum reproducer if possible — `examples/hello_kcore` is a good starting template.

Larger design changes (a new `LatencyModel` semantics, a breaking change to the `Algorithm` interface) should start as a GitHub Discussion or a draft PR with an `RFC:` prefix so we can talk before code is written.

## Code of conduct

Be excellent to each other. Disagreement is fine, hostility isn't.

## License

By submitting a PR you agree to license your contribution under the [MIT License](LICENSE) used by the rest of the project.
