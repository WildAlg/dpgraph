package rrkcore

import "container/heap"

type node struct {
	id  int
	deg float64
	idx int
}

type minHeap []*node

func (h minHeap) Len() int           { return len(h) }
func (h minHeap) Less(i, j int) bool { return h[i].deg < h[j].deg }
func (h minHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].idx = i
	h[j].idx = j
}
func (h *minHeap) Push(x any) {
	n := x.(*node)
	n.idx = len(*h)
	*h = append(*h, n)
}
func (h *minHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.idx = -1
	*h = old[0 : n-1]
	return item
}
func (h *minHeap) update(n *node, deg float64) {
	n.deg = deg
	heap.Fix(h, n.idx)
}
