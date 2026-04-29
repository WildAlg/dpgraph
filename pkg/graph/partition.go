package graph

// Partition splits a graph into numWorkers contiguous shards by vertex ID.
//
// Vertex IDs are assumed to lie in [0, n). Shard i owns vertex IDs in
// [i*chunk, (i+1)*chunk) for i < numWorkers-1, and the final shard absorbs
// the remainder. The returned slice has length numWorkers; each shard's
// AdjacencyList contains only the vertices owned by that shard, with the same
// global vertex IDs preserved as keys.
//
// This is the in-process equivalent of the standalone graph_partitioner.py
// script in the parent paper repo, intended for library users who do not want
// to spawn a Python preprocessor.
func Partition(g *Graph, n, numWorkers int) []*Graph {
	if numWorkers <= 0 {
		return nil
	}
	chunk := n / numWorkers
	shards := make([]*Graph, numWorkers)
	for i := 0; i < numWorkers; i++ {
		lo := i * chunk
		hi := lo + chunk
		if i == numWorkers-1 {
			hi = n
		}
		shard := make(map[int][]int, hi-lo)
		for v := lo; v < hi; v++ {
			if nbrs, ok := g.AdjacencyList[v]; ok {
				shard[v] = nbrs
			} else {
				shard[v] = nil
			}
		}
		shards[i] = &Graph{NumVertices: hi - lo, AdjacencyList: shard}
	}
	return shards
}
