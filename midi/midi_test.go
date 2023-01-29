package midi_test

import (
	"testing"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/rapidmidiex/rmxtui/midi"
	"github.com/rapidmidiex/rmxtui/wsmsg"
	"github.com/stretchr/testify/require"
)

func TestPlayer(t *testing.T) {
	opts := midi.NewPlayerOpts{
		SoundFontName: midi.GeneralUser,
		BufDur:        time.Second * 3,
	}
	player, err := midi.NewPlayer(opts)
	require.NoError(t, err)

	msg := wsmsg.MIDIMsg{
		State:    wsmsg.NOTE_ON,
		Number:   60,
		Velocity: 127,
	}

	// The output buffer (seconds).
	length := int32(1 * 44000)
	streamer := midi.NewMIDIStreamer(length)

	player.Play(msg, streamer)

	// TODO: better test
	require.NotEmpty(t, streamer.Buffers()[0][456])
	require.NotEmpty(t, streamer.Buffers()[1][456])
}

func TestMIDIStreamer(t *testing.T) {
	sr := beep.SampleRate(44100)
	speaker.Init(sr, sr.N(time.Second/5))

	done := make(chan bool)

	speaker.Play(beep.Seq(beep.Take(sr.N(1*time.Second), midi.MidiStreamer{}), beep.Callback(func() {
		done <- true
	})))
	<-done
}
