package sim

import "time"

// LatencyModel maps a (sender, receiver, message-size, round) tuple to a
// simulated network delay. Implementations should be safe to call from many
// goroutines concurrently.
//
// Use dst = -1 to model a worker -> coordinator send.
type LatencyModel interface {
	Delay(src, dst, bytes, round int) time.Duration
}

// Zero applies no latency. Useful for unit tests and for measuring pure
// algorithm cost.
type Zero struct{}

// Delay returns 0.
func (Zero) Delay(int, int, int, int) time.Duration { return 0 }

// BandwidthRTT models each link as a serial channel with a fixed base RTT
// plus bandwidth-dependent transmission time.
//
//	delay = BaseRTT + bytes * 8 / LinkSpeedBitsPerSec
//
// LinkSpeedBitsPerSec must be > 0.
type BandwidthRTT struct {
	LinkSpeedBitsPerSec float64
	BaseRTT             time.Duration
}

// Delay implements LatencyModel.
func (b BandwidthRTT) Delay(_, _, bytes, _ int) time.Duration {
	if b.LinkSpeedBitsPerSec <= 0 {
		return b.BaseRTT
	}
	tx := float64(bytes*8) / b.LinkSpeedBitsPerSec
	return b.BaseRTT + time.Duration(tx*float64(time.Second))
}
