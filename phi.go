package failuredetector

// Port of the Akka PhiAccuralFailureDetector
// https://github.com/akka/akka/blob/master/akka-remote/src/main/scala/akka/remote/PhiAccrualFailureDetector.scala

import (
	"errors"
	"math"
	"sync/atomic"
	"time"
	"unsafe"
)

// PhiAccuralFailureDetector is an implementation of 'The Phi Accrual Failure Detector' by Hayashibara et al. as defined in their paper:
// [http://www.jaist.ac.jp/~defago/files/pdf/IS_RR_2004_010.pdf]
//
// The suspicion level of failure is given by a value called φ (phi).
// The basic idea of the φ failure detector is to express the value of φ on a scale that
// is dynamically adjusted to reflect current network conditions. A configurable
// threshold is used to decide if φ is considered to be a failure.
//
// The value of φ is calculated as:
//
// {{{
// φ = -log10(1 - F(timeSinceLastHeartbeat)
// }}}
// where F is the cumulative distribution function of a normal distribution with mean
// and standard deviation estimated from historical heartbeat inter-arrival times.
//
// This implementation is a port of the fantastic Akka PhiAccrualFailureDetector.
// https://github.com/akka/akka/blob/master/akka-remote/src/main/scala/akka/remote/PhiAccrualFailureDetector.scala
//
// The failure detector is thread-safe and can be safely shared across
// go-routines without additional synchronization.
//
// threshold                - A low threshold is prone to generate many wrong suspicions but ensures a quick detection in the event
//                            of a real crash. Conversely, a high threshold generates fewer mistakes but needs more time to detect
//                            actual crashes
// maxSampleSize            - Number of samples to use for calculation of mean and standard deviation of
//                            inter-arrival times.
// minStdDeviation          - Minimum standard deviation to use for the normal distribution used when calculating phi.
//                            Too low standard deviation might result in too much sensitivity for sudden, but normal, deviations
//                            in heartbeat inter arrival times.
// acceptableHeartbeatPause - Duration corresponding to number of potentially lost/delayed
//                            heartbeats that will be accepted before considering it to be an anomaly.
//                            This margin is important to be able to survive sudden, occasional, pauses in heartbeat
//                            arrivals, due to for example garbage collect or network drop.
// firstHeartbeatEstimate   - Bootstrap the stats with heartbeats that corresponds to
//                            to this duration, with a with rather high standard deviation (since environment is unknown
//                            in the beginning)
type PhiAccuralFailureDetector struct {
	threshold                  float64
	maxSampleSize              uint
	minStdDeviation            time.Duration
	acceptableHeartbeatPause   time.Duration
	firstHeartbeatEstimate     time.Duration
	eventStream                chan<- time.Duration
	firstHeartbeat             heartbeatHistory
	acceptableHeartbeatPauseMS uint64
	minStdDeviationMS          uint64
	state                      *state
}

// state of the PhiAccuralFailureDetector
type state struct {
	history   heartbeatHistory
	timestamp *time.Time
}

// New creates and returns a new failure detector.
func New(
	threshold float64,
	maxSampleSize uint,
	minStdDeviation time.Duration,
	acceptableHeartbeatPause time.Duration,
	firstHeartbeatEstimate time.Duration,
	eventStream chan<- time.Duration) (*PhiAccuralFailureDetector, error) {

	if threshold <= 0.0 {
		return nil, errors.New("threshold must be > 0")
	}

	if maxSampleSize <= 0 {
		return nil, errors.New("maxSampleSize must be > 0")
	}

	if minStdDeviation <= 0 {
		return nil, errors.New("minStdDeviation must be > 0")
	}

	if acceptableHeartbeatPause < 0 {
		return nil, errors.New("acceptableHeartbeatPause must be >= 0")
	}

	if firstHeartbeatEstimate <= 0 {
		return nil, errors.New("heartbeatInterval must be > 0")
	}

	firstHeartbeat := initHeartbeat(maxSampleSize, firstHeartbeatEstimate)

	return &PhiAccuralFailureDetector{
		threshold:                  threshold,
		maxSampleSize:              maxSampleSize,
		minStdDeviation:            minStdDeviation,
		acceptableHeartbeatPause:   acceptableHeartbeatPause,
		firstHeartbeatEstimate:     firstHeartbeatEstimate,
		eventStream:                eventStream,
		firstHeartbeat:             firstHeartbeat,
		acceptableHeartbeatPauseMS: toMillis(acceptableHeartbeatPause),
		minStdDeviationMS:          toMillis(minStdDeviation),
		state:                      &state{history: firstHeartbeat, timestamp: nil},
	}, nil
}

// initHeartbeat returns the initial heartbeat guess
func initHeartbeat(maxSampleSize uint, firstHeartbeatEstimate time.Duration) heartbeatHistory {
	firstHeartbeatEstimateMS := toMillis(firstHeartbeatEstimate)
	stdDeviationMS := firstHeartbeatEstimateMS / 4
	return newHeartbeatHistory(maxSampleSize).
		append(firstHeartbeatEstimateMS - stdDeviationMS).
		append(firstHeartbeatEstimateMS + stdDeviationMS)
}

// Load the state of this failure detector
func (d *PhiAccuralFailureDetector) loadState() *state {
	return (*state)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&d.state))))
}

// CAS the state of this failure detector
func (d *PhiAccuralFailureDetector) casState(old, new *state) bool {
	return atomic.CompareAndSwapPointer(
		(*unsafe.Pointer)(unsafe.Pointer(&d.state)),
		unsafe.Pointer(old),
		unsafe.Pointer(new))
}

// IsAvailable returns true if the resource is considered to be up and healthy; false otherwise.
func (d *PhiAccuralFailureDetector) IsAvailable() bool {
	return d.isAvailableAt(time.Now())
}

func (d *PhiAccuralFailureDetector) isAvailableAt(time time.Time) bool {
	return d.phiAt(time) < d.threshold
}

// IsMonitoring returns true if the failure detectore has received any
// heartbeats and started monitoring of the resource.
func (d *PhiAccuralFailureDetector) IsMonitoring() bool {
	return d.loadState().timestamp != nil
}

// Heartbeat of a monitored resource.
// Notifies the detector that a heartbeat arrived from the monitored resource.
// This causes the detector to update its state.
func (d *PhiAccuralFailureDetector) Heartbeat() {
	for {
		timestamp := time.Now()
		oldState := d.loadState()

		var newHistory heartbeatHistory

		if latestTimestamp := oldState.timestamp; latestTimestamp == nil {
			// this is heartbeat from a new resource
			// add starter records for this new resource
			newHistory = d.firstHeartbeat
		} else {
			// this is a known connection
			interval := timestamp.Sub(*latestTimestamp)
			// don't use the first heartbeat after failure for the history, since a long pause will skew the stats
			if d.isAvailableAt(timestamp) {
				intervalMS := toMillis(interval)
				if intervalMS >= (d.acceptableHeartbeatPauseMS/2) && d.eventStream != nil {
					// heartbeat interval is growing too large (by interval)
					d.eventStream <- interval
				}
				newHistory = oldState.history.append(intervalMS)
			} else {
				newHistory = oldState.history
			}
		}

		newState := &state{history: newHistory, timestamp: &timestamp} // record new timestamp

		// if we won the race then update else try again
		if d.casState(oldState, newState) {
			break
		}
	}
}

// Phi (the suspicion level) of the accrual failure detector.
func (d *PhiAccuralFailureDetector) Phi() float64 {
	return d.phiAt(time.Now())
}

// phiAt a given time of the accrual failure detector.
func (d *PhiAccuralFailureDetector) phiAt(timestamp time.Time) float64 {
	oldState := d.loadState()
	oldTimestamp := oldState.timestamp

	if oldTimestamp == nil {
		return 0.0 // treat unmanaged connections, e.g. with zero heartbeats, as healthy connections
	}

	timeDiff := timestamp.Sub(*oldTimestamp)

	history := oldState.history
	mean := history.mean()
	stdDeviation := d.ensureValidStdDeviation(history.stdDeviation())

	return phi(float64(toMillis(timeDiff)), mean+float64(d.acceptableHeartbeatPauseMS), stdDeviation)
}

func (d *PhiAccuralFailureDetector) ensureValidStdDeviation(stdDeviation float64) float64 {
	return math.Max(stdDeviation, float64(d.minStdDeviationMS))
}

func toMillis(d time.Duration) uint64 {
	return uint64(d.Seconds() * 1e3)
}

func phi(timeDiff, mean, stdDeviation float64) float64 {
	y := (timeDiff - mean) / stdDeviation
	e := math.Exp(-y * (1.5976 + 0.070566*y*y))

	if timeDiff > mean {
		return -math.Log10(e / (1.0 + e))
	}

	return -math.Log10(1.0 - 1.0/(1.0+e))
}
