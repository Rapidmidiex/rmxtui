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

	streamer := midi.NewMIDIStreamer(time.Second)

	player.Play(msg, streamer)

	// TODO: better test
	require.NotEmpty(t, streamer.Buffers()[0][456])
	require.NotEmpty(t, streamer.Buffers()[1][456])
}

func TestMIDIStreamer(t *testing.T) {
	sr := beep.SampleRate(44100)
	bufLen := sr.N(time.Second / 5)
	speaker.Init(sr, bufLen)

	streamer := midi.NewMIDIStreamer(time.Second)

	done := make(chan bool)

	speaker.Play(beep.Seq(beep.Take(sr.N(1*time.Second), streamer), beep.Callback(func() {
		done <- true
	})))
	<-done
}

func TestMIDItoAudio(t *testing.T) {
	sr := beep.SampleRate(44100)
	bufLen := sr.N(time.Second / 5)
	speaker.Init(sr, bufLen)

	player, err := midi.NewPlayer(midi.NewPlayerOpts{
		SoundFontName: midi.GeneralUser,
		BufDur:        time.Second * 1,
	})
	require.NoError(t, err)

	msg := wsmsg.MIDIMsg{
		State:    wsmsg.NOTE_ON,
		Number:   60,
		Velocity: 127,
	}

	streamer := midi.NewMIDIStreamer(time.Second * 2)
	player.Play(msg, streamer)

	done := make(chan bool)

	speaker.Play(beep.Seq(beep.Take(sr.N(1*time.Second), streamer), beep.Callback(func() {
		done <- true
	})))
	<-done
}
