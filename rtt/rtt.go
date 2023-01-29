// Package rtt contains tools for calculating stats on message roundtrip times.
package rtt

import (
	"fmt"
	"math"
	"time"
)

type (
	Stats struct {
		Latest time.Duration
		Avg    time.Duration
		Min    time.Duration
		Max    time.Duration
		Count  int
	}

	Timer struct {
		timestamps map[string]time.Time
	}
)

func NewStats() Stats {
	return Stats{
		Latest: 0,
		Avg:    0,
		Min:    math.MaxInt,
		Max:    -math.MaxInt,
		Count:  0,
	}
}

func NewTimer() *Timer {
	return &Timer{
		timestamps: make(map[string]time.Time),
	}
}

func (t *Timer) Start(msgID string) error {
	_, ok := t.timestamps[msgID]
	if ok {
		return fmt.Errorf("timer already started for ID: %q", msgID)
	}
	t.timestamps[msgID] = time.Now()
	return nil
}

func (t *Timer) Stop(msgID string) time.Duration {
	ts, ok := t.timestamps[msgID]
	if !ok {
		return -1
	}

	delete(t.timestamps, msgID)
	return time.Since(ts)
}

func (s Stats) Calc(d time.Duration) Stats {
	if d < 0 {
		return s
	}
	total := time.Duration(s.Count)
	avg := (total*s.Avg + d) / (total + 1)
	// roundedAvg := math.Round(float64(avg/time.Millisecond)) * float64(time.Millisecond)
	return Stats{
		Latest: d,
		Avg:    avg,
		Min:    time.Duration(math.Min(float64(s.Min), float64(d))),
		Max:    time.Duration(math.Max(float64(s.Max), float64(d))),
		Count:  s.Count + 1,
	}
}
