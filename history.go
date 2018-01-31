package failuredetector

import (
	"math"
)

// heartbeatHistory is a persistent sequence of interval values.
type heartbeatHistory struct {
	maxSampleSize      uint
	intervals          []uint64
	intervalSum        uint64
	squaredIntervalSum uint64
}

// newHeartbeatHistory creates and returns a new heartbeat history object.
func newHeartbeatHistory(maxSampleSize uint) heartbeatHistory {
	return heartbeatHistory{
		maxSampleSize:      maxSampleSize,
		intervals:          make([]uint64, 0, maxSampleSize),
		intervalSum:        0,
		squaredIntervalSum: 0}
}

// mean of this heartbeat history's intervals
func (h heartbeatHistory) mean() float64 {
	return float64(h.intervalSum) / float64(len(h.intervals))
}

// variance of this heartbeat history's intervals
func (h heartbeatHistory) variance() float64 {
	mean := h.mean()
	return (float64(h.squaredIntervalSum) / float64(len(h.intervals))) - (mean * mean)
}

// stdDeviation of this heartbeat history's intervals
func (h heartbeatHistory) stdDeviation() float64 {
	return math.Sqrt(h.variance())
}

// append an interval to the history and return a new object.
// Uses copy on write semantics to ensure heartbeatHistory is persistent.
func (h heartbeatHistory) append(interval uint64) heartbeatHistory {
	var src []uint64
	var droppedInterval uint64

	if uint(len(h.intervals)) < h.maxSampleSize {
		src = h.intervals
	} else {
		droppedInterval = h.intervals[0]
		src = h.intervals[1:]
	}

	dst := make([]uint64, len(src)+1, h.maxSampleSize)
	copy(dst, src)
	dst[len(src)] = interval

	return heartbeatHistory{
		maxSampleSize:      h.maxSampleSize,
		intervals:          dst,
		intervalSum:        h.intervalSum - droppedInterval + interval,
		squaredIntervalSum: h.squaredIntervalSum - (droppedInterval * droppedInterval) + (interval * interval)}
}
