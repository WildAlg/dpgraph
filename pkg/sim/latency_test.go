package sim

import (
	"testing"
	"time"
)

func TestZeroLatency(t *testing.T) {
	if got := (Zero{}).Delay(0, 1, 4096, 0); got != 0 {
		t.Errorf("Zero.Delay = %v, want 0", got)
	}
}

func TestBandwidthRTT(t *testing.T) {
	m := BandwidthRTT{
		LinkSpeedBitsPerSec: 1e6, // 1 Mbps
		BaseRTT:             10 * time.Millisecond,
	}
	// 125 bytes = 1000 bits at 1 Mbps = 1ms transmission, plus 10ms RTT.
	got := m.Delay(0, 1, 125, 0)
	want := 11 * time.Millisecond
	if d := got - want; d < -50*time.Microsecond || d > 50*time.Microsecond {
		t.Errorf("BandwidthRTT.Delay(125B) = %v, want ~%v", got, want)
	}
}

func TestBandwidthRTTZeroSpeed(t *testing.T) {
	m := BandwidthRTT{LinkSpeedBitsPerSec: 0, BaseRTT: 5 * time.Millisecond}
	if got := m.Delay(0, 1, 1024, 0); got != 5*time.Millisecond {
		t.Errorf("zero-speed model should return BaseRTT, got %v", got)
	}
}
