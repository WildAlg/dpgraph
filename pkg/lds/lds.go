// Package lds implements the Levels Data Structure used by hierarchical
// k-core algorithms in dpgraph. Each vertex is assigned a non-negative
// integer level; levels are grouped into "groups" of size levelsPerGroup,
// and the group index of a level is what determines a vertex's effective
// degree threshold in the k-core decomposition.
package lds

import (
	"fmt"
	"math"
	"sync"
)

// GroupDegree returns the degree threshold for the given group index under
// parameter phi: (1+phi)^group.
func GroupDegree(group int, phi float64) float64 {
	return math.Pow(1.0+phi, float64(group))
}

// Vertex stores per-vertex level state. It is exported so algorithms that
// embed an LDS can cheaply walk the underlying slice; in normal use callers
// should prefer the LDS methods, which take the lock when needed.
type Vertex struct {
	Level uint
}

// LDS is a thread-safe array of vertex levels with a fixed size. The lock
// is held by mutating operations; pure reads use the read lock so that
// many workers can sample levels in parallel during a round.
type LDS struct {
	n              int
	levelsPerGroup float64
	lock           sync.RWMutex
	L              []Vertex
}

// New constructs an LDS with n vertices, all initialised to level 0.
func New(n int, levelsPerGroup float64) *LDS {
	L := make([]Vertex, n)
	return &LDS{n: n, levelsPerGroup: levelsPerGroup, L: L}
}

// N returns the number of vertices.
func (l *LDS) N() int { return l.n }

// LevelsPerGroup returns the levels-per-group parameter the LDS was built
// with.
func (l *LDS) LevelsPerGroup() float64 { return l.levelsPerGroup }

// GetLevel returns the level of vertex v, or an error if v is out of range.
func (l *LDS) GetLevel(v uint) (uint, error) {
	if int(v) >= l.n {
		return 0, fmt.Errorf("lds: vertex %d out of bounds (n=%d)", v, l.n)
	}
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.L[v].Level, nil
}

// LevelIncrease bumps the level of vertex v by one.
func (l *LDS) LevelIncrease(v uint) error {
	if int(v) >= l.n {
		return fmt.Errorf("lds: vertex %d out of bounds (n=%d)", v, l.n)
	}
	l.lock.Lock()
	defer l.lock.Unlock()
	l.L[v].Level++
	return nil
}

// GroupForLevel returns the group index of a given level.
func (l *LDS) GroupForLevel(level uint) uint {
	return uint(math.Floor(float64(level) / l.levelsPerGroup))
}
