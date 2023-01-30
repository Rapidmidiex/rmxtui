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

func TestMIDItoAudio(t *testing.T) {
	sr := beep.SampleRate(44100)
	// TODO: Determine buffer length sweet spot.
	// Bigger -> less CPU, slower response
	// Lower -> more CPU, faster response
	bufLen := sr.N(time.Millisecond * 20)
	speaker.Init(sr, bufLen)

	player, err := midi.NewPlayer(midi.NewPlayerOpts{
		SoundFontName: midi.GeneralUser,
	})
	require.NoError(t, err)

	msg := wsmsg.MIDIMsg{
		State:    wsmsg.NOTE_ON,
		Number:   70, // D5
		Velocity: 127,
	}

	duration := time.Second * 5
	streamer := midi.NewMIDIStreamer(duration)
	player.Play(msg, streamer)

	done := make(chan bool)

	speaker.Play(beep.Seq(
		beep.Take(sr.N(duration), streamer),
		beep.Callback(func() {
			done <- true
		})))
	<-done
}
