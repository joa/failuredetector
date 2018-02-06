package failuredetector

import "time"

type clock func() time.Time

func defaultClock() time.Time {
	return time.Now()
}

type fakeClock struct {
	index int
	times []time.Time
}

func (c *fakeClock) apply() time.Time {
	r := c.times[c.index]
	c.index++
	return r
}

func newFakeClock(intervals []int) clock {
	var t time.Time
	r := make([]time.Time, len(intervals))

	for i, v := range intervals {
		t = t.Add(time.Duration(v) * time.Millisecond)
		r[i] = t
	}

	c := &fakeClock{index: 0, times: r}

	return c.apply
}
