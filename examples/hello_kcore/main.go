// hello_kcore is the smallest possible end-to-end dpgraph program: load a
// 10-vertex toy graph, run the k-core LDP algorithm with simulated network
// latency, and print the per-vertex core-number estimates plus a metrics
// summary.
//
// Run from the dpgraph repo root:
//
//	go run ./examples/hello_kcore
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/WildAlg/dpgraph/pkg/algo"
	"github.com/WildAlg/dpgraph/pkg/sim"

	_ "github.com/WildAlg/dpgraph/algos/kcoreldp"
)

func main() {
	tmp, err := os.MkdirTemp("", "dpgraph-hello-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	cfg := sim.Config{
		Graph:      "toy",
		GraphPath:  "testdata/toy_10node_adj",
		GraphSize:  10,
		AlgoName:   "kcoreLDP",
		NumWorkers: 2,
		Epsilon:    1.0,
		Phi:        0.5,
		Factor:     0.8,
		Bias:       false,
		BiasFactor: 1,
		Noise:      false,
		Runs:       1,
		RunID:      0,
		Seed:       42,
		OutputDir:  tmp,
		OutputTag:  "hello",
		Latency: sim.BandwidthRTT{
			LinkSpeedBitsPerSec: 25e6, // 25 Mbps, matching the original triangle-counting LDP stub
			BaseRTT:             1 * time.Millisecond,
		},
	}

	if err := algo.Run(cfg); err != nil {
		log.Fatal(err)
	}

	resultPath := tmp + "/" + sim.FilenameForConfig(cfg)
	body, err := os.ReadFile(resultPath)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("=== core numbers ===")
	fmt.Println(string(body))
	metricsBody, _ := os.ReadFile(resultPath + ".metrics.json")
	fmt.Println("=== metrics ===")
	fmt.Println(string(metricsBody))
}
