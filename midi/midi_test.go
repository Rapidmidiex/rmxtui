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
	noteDuration := time.Second * 5
	speaker.Init(sr, bufLen)
	synth, err := midi.NewSynth(midi.NewSynthOpts{
		SoundFontName: midi.GeneralUser,
	})
	require.NoError(t, err)

	t.Run("converts an RMX MIDI message to stereo sound", func(t *testing.T) {
		msg := wsmsg.MIDIMsg{
			State:    wsmsg.NOTE_ON,
			Number:   70, // Bb4
			Velocity: 127,
		}

		streamer := midi.NewMIDIStreamer(noteDuration)
		synth.Render(msg, streamer)

		done := make(chan bool)

		speaker.Play(beep.Seq(
			beep.Take(sr.N(noteDuration), streamer),
			// callback called after the above streamer is complete, since we're using Seq().
			beep.Callback(func() {
				done <- true
			}),
		))

		// Wait for the callback streamer to run
		<-done
	})

	t.Run("plays multiple notes at once", func(t *testing.T) {
		streamer1 := midi.NewMIDIStreamer(noteDuration)
		synth.Render(wsmsg.MIDIMsg{
			State:    wsmsg.NOTE_ON,
			Number:   70, // Bb4
			Velocity: 127,
		}, streamer1)

		streamer2 := midi.NewMIDIStreamer(noteDuration)
		synth.Render(wsmsg.MIDIMsg{
			State:    wsmsg.NOTE_ON,
			Number:   67, // G4
			Velocity: 127,
		}, streamer2)

		streamer3 := midi.NewMIDIStreamer(noteDuration)
		synth.Render(wsmsg.MIDIMsg{
			State:    wsmsg.NOTE_ON,
			Number:   76, // E5
			Velocity: 108,
		}, streamer3)

		mixer := beep.Mixer{}
		mixer.Add(
			beep.Take(sr.N(noteDuration), streamer1),
			beep.Take(sr.N(noteDuration), streamer2),
		)

		// Only need to call Play once.
		// The mixer will play silence if the streamers are drained.
		speaker.Play(&mixer)

		// Add another note after a pause
		time.Sleep(time.Millisecond * 300)
		mixer.Add(
			beep.Take(sr.N(noteDuration), streamer3),
		)

		// Wait for speaker to finish playing
		// (Callback does not work with the mixer)
		time.Sleep(noteDuration + time.Second/2)
	})

	t.Run("plays chords", func(t *testing.T) {
		streamer1 := midi.NewMIDIStreamer(noteDuration)
		synth.Render(wsmsg.MIDIMsg{
			State:    wsmsg.NOTE_ON,
			Number:   40,
			Velocity: 127,
		}, streamer1)

		streamer2 := midi.NewMIDIStreamer(noteDuration)
		synth.Render(wsmsg.MIDIMsg{
			State:    wsmsg.NOTE_ON,
			Number:   42,
			Velocity: 108,
		}, streamer2)

		mixer := beep.Mixer{}
		mixer.Add(
			beep.Take(sr.N(noteDuration), streamer1),
			beep.Take(sr.N(noteDuration), streamer2),
		)

		// Only need to call Play once.
		// The mixer will play silence if the streamers are drained.
		speaker.Play(&mixer)

		// Play another note concurrently, ie a chord.
		go func(streamer beep.Streamer) {
			mixer.Add(
				beep.Take(sr.N(noteDuration), streamer2),
			)
		}(streamer2)

		// Wait for speaker to finish playing
		// (Callback does not work with the mixer)
		time.Sleep(noteDuration + time.Millisecond*100)
	})
}
