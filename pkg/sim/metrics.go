package sim

import (
	"encoding/json"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics records timing and message statistics for a single run. It is safe
// for concurrent use; algorithm authors do not need to lock around it.
//
// Per-phase wall times (e.g. "setup", "algo") live in the phases map and
// are accumulated via PhaseTimer. Top-level WallTime is the total
// start-to-finish elapsed time set by Finish.
type Metrics struct {
	StartedAt        time.Time     `json:"started_at"`
	FinishedAt       time.Time     `json:"finished_at"`
	WallTime         time.Duration `json:"wall_time_ns"`
	TotalMessages    int64         `json:"total_messages"`
	TotalBytes       int64         `json:"total_bytes"`
	SimulatedNetwork time.Duration `json:"simulated_network_ns"`
	Rounds           int           `json:"rounds"`

	mu          sync.Mutex
	phaseTimers map[string]time.Duration
}

// NewMetrics returns a fresh Metrics ready to record a run.
func NewMetrics() *Metrics {
	return &Metrics{
		StartedAt:   time.Now(),
		phaseTimers: make(map[string]time.Duration),
	}
}

// PhaseTimer marks the start of a named phase; call the returned function
// when the phase ends to accumulate wall-clock time under that name.
func (m *Metrics) PhaseTimer(name string) func() {
	start := time.Now()
	return func() {
		d := time.Since(start)
		m.mu.Lock()
		m.phaseTimers[name] += d
		m.mu.Unlock()
	}
}

func (m *Metrics) recordSend(bytes int, delay time.Duration) {
	atomic.AddInt64(&m.TotalMessages, 1)
	atomic.AddInt64(&m.TotalBytes, int64(bytes))
	// SimulatedNetwork is a sum across worker goroutines; readers should
	// note that workers' sleeps may overlap in wall-clock time.
	atomic.AddInt64((*int64)(&m.SimulatedNetwork), int64(delay))
}

// Finish stamps FinishedAt and computes WallTime.
func (m *Metrics) Finish() {
	m.FinishedAt = time.Now()
	m.WallTime = m.FinishedAt.Sub(m.StartedAt)
}

// WriteJSON serialises the metrics object as JSON.
func (m *Metrics) WriteJSON(w io.Writer) error {
	m.mu.Lock()
	phases := make(map[string]int64, len(m.phaseTimers))
	for k, v := range m.phaseTimers {
		phases[k+"_ns"] = int64(v)
	}
	m.mu.Unlock()
	type out struct {
		*Metrics
		Phases map[string]int64 `json:"phases"`
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out{Metrics: m, Phases: phases})
}
