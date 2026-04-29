package sim

import "sync"

// State is a typed key/value store shared between the coordinator and all
// workers in a run. Algorithms use it to publish per-round broadcast
// information (e.g. an LDS, a noisy threshold, a partial aggregate) without
// each algorithm having to roll its own RWMutex.
//
// Reads are concurrent-safe under the read lock; writes take the write lock.
// Writes do NOT trigger latency injection — callers that want to model the
// cost of a broadcast should call WorkerCtx.SimulateSend in the receiving
// worker.
type State struct {
	mu sync.RWMutex
	m  map[string]any
}

// NewState returns an empty State.
func NewState() *State { return &State{m: make(map[string]any)} }

// Set stores v under key.
func (s *State) Set(key string, v any) {
	s.mu.Lock()
	s.m[key] = v
	s.mu.Unlock()
}

// Get returns the value stored under key and a presence flag.
func (s *State) Get(key string) (any, bool) {
	s.mu.RLock()
	v, ok := s.m[key]
	s.mu.RUnlock()
	return v, ok
}
