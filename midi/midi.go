package midi

import (
	"embed"
	"fmt"
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
	}

	SoundFontName int

	NewPlayerOpts struct {
		// Name of SoundFont to use for the synthesizer.
		SoundFontName SoundFontName
	}

	MidiStreamer struct {
		pos   int
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
	}, nil
}

// Play synthesizes the given MIDI note and write the audio data to the streamer's left/right buffers.
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

func NewMIDIStreamer(clipLength time.Duration) *MidiStreamer {
	// TODO: Get sample rate from config or struct
	bufLen := int(44100 * clipLength.Seconds())
	return &MidiStreamer{
		left:  make([]float32, bufLen),
		right: make([]float32, bufLen),
	}
}

// Stream implements beep.Streamer.
func (ms *MidiStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	left := make([]float32, len(samples))
	right := make([]float32, len(samples))

	ms.Read(left, right)

	for i := range samples {
		samples[i][0] = float64(left[i])
		samples[i][1] = float64(right[i])
	}
	return len(samples), true
}

// Len returns the total number of samples of the Streamer.
func (ms MidiStreamer) Len() int {
	// left and right have the same length
	return int(len(ms.left))
}

// Position returns the current position of the Streamer.
func (ms MidiStreamer) Position() int {
	return ms.pos
}

// Seek sets the position of the Streamer to the provided value.
func (ms *MidiStreamer) Seek(p int) error {
	if p < 0 || p > len(ms.left) {
		return fmt.Errorf("p is out of range: %d", p)
	}
	ms.pos = p
	return nil
}

func (ms MidiStreamer) Err() error {
	return nil
}

// Read reads from the MIDIStreamer's left/right buffers at the current Pos and writes the contents to the out []float32 buffers.
func (ms *MidiStreamer) Read(outLeft, outRight []float32) (int, error) {
	nRead := 0
	for i := range outLeft {
		readPos := i + ms.pos
		if readPos > len(ms.left) {
			return nRead, fmt.Errorf("index is out of range: %d", readPos)
		}
		outLeft[i] = ms.left[readPos]
		outRight[i] = ms.right[readPos]
		nRead++
	}
	ms.pos += nRead
	return nRead, nil
}
