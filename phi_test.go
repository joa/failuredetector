package failuredetector

import (
	"sync"
	"testing"
	"time"
)

func createFailureDetector() *PhiAccuralFailureDetector {
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

	return res
}

func TestStress(t *testing.T) {
	// this is not a unit test - just verify that the history and atomic handling
	// isn't totally bonkers

	d := createFailureDetector()
	var wg sync.WaitGroup

	// ensure we're monitoring so the test is never flaky
	d.Heartbeat()

	if !verifyMonitoringAndAvailable(t, d) {
		return
	}

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			for i := 0; i < 1000; i++ {
				d.Heartbeat()
			}
			wg.Done()
		}()
	}

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			for i := 0; i < 1000; i++ {
				if !verifyMonitoringAndAvailable(t, d) {
					break
				}
			}
			wg.Done()
		}()
	}

	wg.Wait()

	verifyMonitoringAndAvailable(t, d)
}

func verifyMonitoringAndAvailable(t *testing.T, d *PhiAccuralFailureDetector) bool {
	if !d.IsAvailable() {
		t.Error("detector should report resource available")
		return false
	}

	if !d.IsMonitoring() {
		t.Error("detector should be monitoring")
		return false
	}

	return true
}
