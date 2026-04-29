package graph

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndPartition(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "small_adj")
	if err := os.WriteFile(path, []byte("0 1\n1 0\n2 3\n3 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	g, err := Load(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if g.NumVertices != 4 {
		t.Errorf("NumVertices=%d, want 4", g.NumVertices)
	}
	shards := Partition(g, 4, 2)
	if len(shards) != 2 {
		t.Fatalf("len(shards)=%d, want 2", len(shards))
	}
	for w, s := range shards {
		if len(s.AdjacencyList) != 2 {
			t.Errorf("shard %d: %d vertices, want 2", w, len(s.AdjacencyList))
		}
	}
}

func TestLoadBidirectional(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "uni_adj")
	if err := os.WriteFile(path, []byte("0 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	g, err := Load(path, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.AdjacencyList[0]) != 1 || g.AdjacencyList[0][0] != 1 {
		t.Errorf("0->1 missing: %v", g.AdjacencyList)
	}
	if len(g.AdjacencyList[1]) != 1 || g.AdjacencyList[1][0] != 0 {
		t.Errorf("1->0 missing (bidirectional): %v", g.AdjacencyList)
	}
}
