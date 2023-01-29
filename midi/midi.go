package midi

import (
	"embed"
	"math/rand"
	"path"
	"time"

	"github.com/rapidmidiex/rmxtui/wsmsg"
	"github.com/sinshu/go-meltysynth/meltysynth"
)

//go:embed sound_fonts
var soundFontsFS embed.FS

const (
	GeneralUser SoundFontName = iota
)

type (
	Player struct {
		// SoundFonts available in the embedded FS.
		soundFontPaths map[SoundFontName]string
		synth          *meltysynth.Synthesizer
		synthSettings  *meltysynth.SynthesizerSettings
		bufSecs        int32
	}

	SoundFontName int

	NewPlayerOpts struct {
		// Name of SoundFont to use for the synthesizer.
		SoundFontName SoundFontName
		// Audio buffer length.
		BufDur time.Duration
	}

	MidiStreamer struct {
		left  []float32
		right []float32
	}
)

func NewPlayer(o NewPlayerOpts) (Player, error) {
	soundFonts := map[SoundFontName]string{
		GeneralUser: "GeneralUser_GS_MuseScore_v1.442.sf2",
		// TODO: Add more as needed
		// https://musescore.org/en/handbook/3/soundfonts-and-sfz-files#list
	}

	// Load the SoundFont.
	sf2, err := soundFontsFS.Open(
		path.Join("sound_fonts", soundFonts[o.SoundFontName]),
	)
	if err != nil {
		return Player{}, err
	}
	soundFont, _ := meltysynth.NewSoundFont(sf2)
	sf2.Close()

	// Create the synthesizer.
	settings := meltysynth.NewSynthesizerSettings(44100)
	synthesizer, _ := meltysynth.NewSynthesizer(soundFont, settings)

	return Player{
		soundFontPaths: soundFonts,
		synth:          synthesizer,
		synthSettings:  settings,
		bufSecs:        int32(o.BufDur.Seconds()),
	}, nil
}

func (p Player) Play(msg wsmsg.MIDIMsg, streamer *MidiStreamer) {
	note := int32(msg.Number)
	vel := int32(msg.Velocity)

	switch msg.State {
	case wsmsg.NOTE_ON:
		p.synth.NoteOn(0, note, vel)
	case wsmsg.NOTE_OFF:
		p.synth.NoteOff(0, note)
	}

	// Render the waveform.
	p.synth.Render(streamer.left, streamer.right)
}

func NewMIDIStreamer(bufLen int32) *MidiStreamer {
	return &MidiStreamer{
		left:  make([]float32, bufLen),
		right: make([]float32, bufLen),
	}
}

// Stream implements beep.Streamer
func (ms MidiStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	for i := range samples {
		// TODO: Set samples from rendered MIDI
		// samples[i][0] = float64(ms.left[i])
		// samples[i][1] = float64(ms.right[i])

		// TODO: Right now, this is a noise generator
		samples[i][0] = rand.Float64()*2 - 1
		samples[i][1] = rand.Float64()*2 - 1
	}
	return len(samples), true
}

func (ms MidiStreamer) Err() error {
	return nil
}

func (ms MidiStreamer) Buffers() [2][]float32 {
	return [2][]float32{ms.left, ms.right}
}
