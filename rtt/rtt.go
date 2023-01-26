// Package rtt contains tools for calculating stats on message roundtrip times.
package rtt

import (
	"math"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type (
	CalcMsg struct {
		Latest time.Duration
		Avg    time.Duration
		Min    time.Duration
		Max    time.Duration
	}
)

func CalcStats(ping time.Duration, prev []time.Duration) tea.Cmd {
	roundedAvg := math.Round(float64(Avg(prev)/time.Millisecond)) * float64(time.Millisecond)
	return func() tea.Msg {
		return CalcMsg{
			Latest: ping,
			Avg:    time.Duration(roundedAvg),
			Max:    Max(prev),
			Min:    Min(prev),
		}
	}
}

func Min(times []time.Duration) time.Duration {
	min := math.Inf(1)
	for _, t := range times {
		min = math.Min(min, float64(t))
	}
	return time.Duration(min)
}

func Max(times []time.Duration) time.Duration {
	max := math.Inf(-1)
	for _, t := range times {
		max = math.Max(max, float64(t))
	}
	return time.Duration(max)
}

func Avg(times []time.Duration) time.Duration {
	if len(times) == 0 {
		return 0
	}
	sum := time.Duration(0)
	for _, t := range times {
		sum = sum + t
	}
	return sum / time.Duration(len(times))
}
