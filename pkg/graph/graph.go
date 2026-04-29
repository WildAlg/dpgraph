// Package graph provides the in-memory graph type used across dpgraph and a
// loader that reads the simple "u v" edge-list format the framework's example
// graphs use.
package graph

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Graph is an adjacency-list graph keyed by integer vertex IDs.
//
// AdjacencyList is intentionally a map (not a slice) so that vertices may be
// non-contiguous; callers that assume dense IDs can read NumVertices and use
// the convention that IDs run [0, NumVertices).
type Graph struct {
	NumVertices   int
	AdjacencyList map[int][]int
}

// Load reads an edge-list adjacency file. Each line is "u v" (whitespace
// separated). If bidirectional is true, every edge is added in both
// directions; otherwise only u -> v is recorded.
//
// Lines whose neighbour value is negative are treated as isolated-vertex
// markers (the vertex is recorded with no outgoing edge).
func Load(filename string, bidirectional bool) (*Graph, error) {
	adj := make(map[int][]int)

	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("graph: open %s: %w", filename, err)
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	// Allow long lines for high-degree vertices.
	s.Buffer(make([]byte, 1<<20), 1<<24)
	lineNum := 0
	for s.Scan() {
		lineNum++
		fields := strings.Fields(s.Text())
		if len(fields) < 2 {
			continue
		}
		u, err := strconv.Atoi(fields[0])
		if err != nil {
			return nil, fmt.Errorf("graph: line %d: parse vertex: %w", lineNum, err)
		}
		v, err := strconv.Atoi(fields[1])
		if err != nil {
			return nil, fmt.Errorf("graph: line %d: parse neighbour: %w", lineNum, err)
		}
		if v >= 0 {
			adj[u] = append(adj[u], v)
		} else if _, ok := adj[u]; !ok {
			adj[u] = nil
		}
		if bidirectional && v >= 0 {
			adj[v] = append(adj[v], u)
		}
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("graph: scan %s: %w", filename, err)
	}

	return &Graph{
		NumVertices:   len(adj),
		AdjacencyList: adj,
	}, nil
}
