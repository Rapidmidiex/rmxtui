package rtt_test

import (
	"testing"
	"time"

	"github.com/rapidmidiex/rmxtui/rtt"
	"github.com/stretchr/testify/require"
)

func TestStats(t *testing.T) {
	prevPings := []time.Duration{
		time.Millisecond * 19,
		time.Millisecond * 1000,
		time.Millisecond * 129,
		time.Millisecond * 34,
		time.Millisecond * 36,
		time.Millisecond * 49,
		time.Millisecond * 234,
	}

	stats := rtt.NewStats()

	for _, p := range prevPings {
		stats = stats.Calc(p)
	}

	want := rtt.Stats{
		Min:    time.Millisecond * 19,
		Max:    time.Millisecond * 1000,
		Avg:    214428570, // 214.42857142857 (ms) rounded to nearest ns
		Latest: time.Millisecond * 234,
		Count:  7,
	}
	require.Equal(t, want, stats)
}
