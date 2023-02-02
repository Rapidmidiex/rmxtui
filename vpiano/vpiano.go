package vpiano

type (
	Note struct {
		// MIDI note number, based on C4=60
		MIDI int
		// Name of the note, ex: "C", "F#Gb"
		Name string
		// Denotes if note is sharp/flat ie. "black" key.
		IsAccidental bool
		// qwerty keyboard key binding.
		KeyBinding string
	}

	Notes []Note

	NoteKeyMap map[string]Note

	octave int
)

const (
	Cneg2 octave = iota - 2
	Cneg1
	C0
	C1
	C2
	C3
	C4
	C5
	C6
	C7
)

var noteNames = []struct {
	name         string
	isAccidental bool
}{
	{name: "A", isAccidental: false},
	{name: "A#/Bb", isAccidental: true},
	{name: "B", isAccidental: false},
	{name: "C", isAccidental: false},
	{name: "C#/Db", isAccidental: true},
	{name: "D", isAccidental: false},
	{name: "D#/Eb", isAccidental: true},
	{name: "E", isAccidental: false},
	{name: "F", isAccidental: false},
	{name: "F#/Gb", isAccidental: true},
	{name: "G", isAccidental: false},
	{name: "G#/Ab", isAccidental: true}}

// MakeOctaveNotes creates list of piano note, MIDI #, qwerty keyboard bindings given an octave name, for example "C4". The keybindings start a C, using the home row for naturals and q-row for accidentals, in an attempt to map close to actual piano fingerings.
func MakeOctaveNotes(octave octave) Notes {
	// qwerty keys ordered to allow for fingering similar to a real piano.
	qwertyKeys := []string{"a", "w", "s", "e", "d", "f", "t", "g", "y", "h", "u", "j", "k", "o", "l", "p", ";", "'"}
	// MIDI number for C0
	midiC0 := 12
	// # of available qwerty keys to map to notes.
	keyboardLen := 18
	octaveLen := 12
	notes := make([]Note, 0)

	for i := 0; i < keyboardLen; i++ {
		k := noteNames[(i+3)%octaveLen]
		midi := midiC0 + (octaveLen * int(octave)) + i
		kb := qwertyKeys[i]

		note := Note{
			MIDI:         midi,
			Name:         k.name,
			IsAccidental: k.isAccidental,
			KeyBinding:   kb,
		}
		notes = append(notes, note)
	}

	return notes
}

func (notes Notes) ToBindingMap() NoteKeyMap {
	nMap := make(NoteKeyMap, 0)
	for _, n := range notes {
		nMap[n.KeyBinding] = n
	}
	return nMap
}

func InRange(midiNum int) bool {
	return midiNum > 20 && midiNum < 128
}
