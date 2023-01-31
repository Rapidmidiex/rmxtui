package vpiano_test

import (
	"testing"

	"github.com/rapidmidiex/rmxtui/vpiano"
	"github.com/stretchr/testify/require"
)

func TestMakeOctaveNotes(t *testing.T) {
	got := vpiano.MakeOctaveNotes(vpiano.C4)
	wantNotes := []vpiano.Note{
		{MIDI: 60, KeyBinding: "a", Name: "C", IsAccidental: false},
		{MIDI: 61, KeyBinding: "w", Name: "C#/Db", IsAccidental: true},
		{MIDI: 62, KeyBinding: "s", Name: "D", IsAccidental: false},
		{MIDI: 63, KeyBinding: "e", Name: "D#/Eb", IsAccidental: true},
		{MIDI: 64, KeyBinding: "d", Name: "E", IsAccidental: false},
		{MIDI: 65, KeyBinding: "f", Name: "F", IsAccidental: false},
		{MIDI: 66, KeyBinding: "t", Name: "F#/Gb", IsAccidental: true},
		{MIDI: 67, KeyBinding: "g", Name: "G", IsAccidental: false},
		{MIDI: 68, KeyBinding: "y", Name: "G#/Ab", IsAccidental: true},
		{MIDI: 69, KeyBinding: "h", Name: "A", IsAccidental: false},
		{MIDI: 70, KeyBinding: "u", Name: "A#/Bb", IsAccidental: true},
		{MIDI: 71, KeyBinding: "j", Name: "B", IsAccidental: false},
		{MIDI: 72, KeyBinding: "k", Name: "C", IsAccidental: false},
		{MIDI: 73, KeyBinding: "o", Name: "C#/Db", IsAccidental: true},
		{MIDI: 74, KeyBinding: "l", Name: "D", IsAccidental: false},
		{MIDI: 75, KeyBinding: "p", Name: "D#/Eb", IsAccidental: true},
		{MIDI: 76, KeyBinding: ";", Name: "E", IsAccidental: false},
		{MIDI: 77, KeyBinding: "'", Name: "F", IsAccidental: false},
	}

	for i, want := range wantNotes {
		require.Equal(t, want, got[i])
	}
}
