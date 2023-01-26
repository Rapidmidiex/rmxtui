package ping_test

import (
	"testing"
	"time"

	"github.com/rapidmidiex/rmxtui/ping"
	"github.com/stretchr/testify/require"
)

func TestCalc(t *testing.T) {
	prevPings := []time.Duration{
		time.Millisecond * 19,
		time.Millisecond * 1000,
		time.Millisecond * 129,
		time.Millisecond * 34,
		time.Millisecond * 36,
		time.Millisecond * 49,
		time.Millisecond * 234,
	}

	gotCmd := ping.CalcStats(time.Millisecond*30, prevPings)
	want := ping.CalcMsg{
		Min:    time.Millisecond * 19,
		Max:    time.Millisecond * 1000,
		Avg:    time.Millisecond * 214, // 214.42857142857 rounded to nearest ms
		Latest: time.Millisecond * 30,
	}
	require.Equal(t, want, gotCmd())
}
