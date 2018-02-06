package failuredetector

import (
	"testing"
	"time"
)

func TestFakeClock(t *testing.T) {
	var clockTests = []struct {
		intervals []int
	}{
		{[]int{0, 1, 2, 3}},
		{[]int{1000, 100, 200, 300}},
	}

	for _, tt := range clockTests {
		fc := newFakeClock(tt.intervals)

		last := fc()

		for _, i := range tt.intervals[1:] {
			d := fc()
			if toMillis(d.Sub(last)) != uint64(i) {
				t.Errorf("FakeClock(%v) => %v, want %v", tt.intervals, d.Sub(last), time.Duration(i)*time.Millisecond)
			}
			last = d
		}

	}
}
