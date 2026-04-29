package lds

import (
	"sync"
	"testing"
)

func TestLevelIncreaseAndGet(t *testing.T) {
	l := New(4, 1.5)
	if v, _ := l.GetLevel(0); v != 0 {
		t.Errorf("initial level = %d, want 0", v)
	}
	if err := l.LevelIncrease(2); err != nil {
		t.Fatal(err)
	}
	if v, _ := l.GetLevel(2); v != 1 {
		t.Errorf("after Increase: level = %d, want 1", v)
	}
}

func TestLevelOutOfRange(t *testing.T) {
	l := New(2, 1.0)
	if _, err := l.GetLevel(5); err == nil {
		t.Error("expected error for out-of-range vertex")
	}
}

func TestConcurrentIncrement(t *testing.T) {
	l := New(1, 1.0)
	var wg sync.WaitGroup
	const n = 1000
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = l.LevelIncrease(0)
		}()
	}
	wg.Wait()
	if v, _ := l.GetLevel(0); v != n {
		t.Errorf("after %d concurrent increments, level=%d", n, v)
	}
}

func TestGroupForLevel(t *testing.T) {
	l := New(1, 4.0)
	cases := map[uint]uint{0: 0, 3: 0, 4: 1, 7: 1, 8: 2}
	for in, want := range cases {
		if got := l.GroupForLevel(in); got != want {
			t.Errorf("GroupForLevel(%d)=%d, want %d", in, got, want)
		}
	}
}
