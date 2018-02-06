package failuredetector

import (
	"testing"
	"time"
)

func createFailureDetector(c clock) *PhiAccuralFailureDetector {
	res, err := New(
		8.0,
		1000,
		10*time.Millisecond,
		0,
		1*time.Second,
		nil)

	if err != nil {
		panic(err)
	}

	res.clock = c

	return res
}

func TestNodeIsAvailableAfterASeriesOfSuccessfulHeartbeats(t *testing.T) {
	timeInterval := []int{0, 1000, 100, 100}
	fd := createFailureDetector(newFakeClock(timeInterval))

	fd.Heartbeat()
	fd.Heartbeat()
	fd.Heartbeat()

	if !fd.IsAvailable() {
		t.Error("detector should report resource available")
	}
}

func TestNodeMarkedDeadAfterHeartbeatsAreMissed(t *testing.T) {
	timeInterval := []int{0, 1000, 100, 100, 4000, 3000}
	fd := createFailureDetector(newFakeClock(timeInterval))
	fd.threshold = 3.0

	fd.Heartbeat() //0
	fd.Heartbeat() //1000
	fd.Heartbeat() //1100

	if !fd.IsAvailable() { //1200
		t.Error("detector should report resource available")
	}

	fd.clock() //5200, but unrelated resource

	if fd.IsAvailable() { //1200
		t.Error("detector shouldn't report resource available")
	}
}
