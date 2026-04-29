// Command dpgraph runs a registered algorithm against a graph using a YAML
// config compatible with the parent paper repo's experiments/configs/*.yaml.
//
//	dpgraph -config_file mygraph-kcoreLDP.yaml -workers 8 -graph_dir ./graphs
//
// Output files use the legacy schema so scripts/get_results.py keeps working;
// a sibling .metrics.json is also written.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/WildAlg/dpgraph/pkg/algo"
	"github.com/WildAlg/dpgraph/pkg/sim"

	// Side-effect imports register algorithms with the package registry.
	_ "github.com/WildAlg/dpgraph/algos/kcorecdp"
	_ "github.com/WildAlg/dpgraph/algos/kcoreldp"
	_ "github.com/WildAlg/dpgraph/algos/rrkcore"
	_ "github.com/WildAlg/dpgraph/algos/rrtcount"
	_ "github.com/WildAlg/dpgraph/algos/tcountcdp"
	_ "github.com/WildAlg/dpgraph/algos/tcountldp"
)

type yamlConfig struct {
	Graph         string  `yaml:"graph"`
	GraphSize     int     `yaml:"graph_size"`
	AlgoName      string  `yaml:"algo_name"`
	NumWorkers    int     `yaml:"num_workers"`
	Epsilon       float64 `yaml:"epsilon"`
	Phi           float64 `yaml:"phi"`
	Factor        float64 `yaml:"factor"`
	Bias          bool    `yaml:"bias"`
	BiasFactor    int     `yaml:"bias_factor"`
	Noise         bool    `yaml:"noise"`
	Runs          int     `yaml:"runs"`
	OutputTag     string  `yaml:"output_file_tag"`
	GraphLoc      string  `yaml:"graph_loc"`
	Bidirectional bool    `yaml:"bidirectional"`
}

func main() {
	var (
		configFile    = flag.String("config_file", "", "path to YAML config (required)")
		workers       = flag.Int("workers", 0, "override num_workers from YAML (0 = use YAML)")
		seed          = flag.Int64("seed", 1, "RNG seed (per-worker streams derived from this)")
		linkSpeedMbps = flag.Float64("link_mbps", 0, "simulated link speed in Mbps; 0 disables latency injection")
		baseRTTMillis = flag.Float64("rtt_ms", 0, "simulated base RTT in milliseconds")
		outputDir     = flag.String("output_dir", ".", "directory to write result and metrics files into")
		runID         = flag.Int("run_id", 0, "run identifier folded into the output filename")
	)
	flag.Parse()

	if *configFile == "" {
		fmt.Fprintln(os.Stderr, "usage: dpgraph -config_file <file.yaml> [-workers N] [-link_mbps M] [-rtt_ms R]")
		fmt.Fprintln(os.Stderr, "registered algorithms:", algo.Names())
		os.Exit(2)
	}

	cfg, err := loadYAML(*configFile)
	if err != nil {
		log.Fatalf("dpgraph: %v", err)
	}

	if *workers > 0 {
		cfg.NumWorkers = *workers
	}
	if cfg.NumWorkers <= 0 {
		cfg.NumWorkers = 1
	}
	if cfg.Factor == 0 {
		cfg.Factor = 0.8
	}
	if cfg.Epsilon == 0 {
		cfg.Epsilon = 1.0
	}

	graphPath := filepath.Join(cfg.GraphLoc, cfg.Graph+"_adj")

	var latency sim.LatencyModel = sim.Zero{}
	if *linkSpeedMbps > 0 || *baseRTTMillis > 0 {
		latency = sim.BandwidthRTT{
			LinkSpeedBitsPerSec: *linkSpeedMbps * 1e6,
			BaseRTT:             time.Duration(*baseRTTMillis * float64(time.Millisecond)),
		}
	}

	runCfg := sim.Config{
		Graph:         cfg.Graph,
		GraphPath:     graphPath,
		GraphSize:     cfg.GraphSize,
		AlgoName:      cfg.AlgoName,
		NumWorkers:    cfg.NumWorkers,
		Epsilon:       cfg.Epsilon,
		Phi:           cfg.Phi,
		Factor:        cfg.Factor,
		Bias:          cfg.Bias,
		BiasFactor:    cfg.BiasFactor,
		Noise:         cfg.Noise,
		Runs:          cfg.Runs,
		RunID:         *runID,
		Seed:          *seed,
		Bidirectional: cfg.Bidirectional,
		OutputDir:     *outputDir,
		OutputTag:     cfg.OutputTag,
		Latency:       latency,
	}

	if err := algo.Run(runCfg); err != nil {
		log.Fatalf("dpgraph: %v", err)
	}
}

func loadYAML(path string) (*yamlConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var c yamlConfig
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &c, nil
}
