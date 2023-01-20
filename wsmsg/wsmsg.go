package wsmsg

type (
	MsgType   int
	NoteState int

	TextMsg struct {
		Typ     MsgType `json:"type"`
		Payload string  `json:"payload"`
	}

	MIDIMsg struct {
		State  NoteState `json:"state"`
		Number int       `json:"number"`
	}
)

const (
	TEXT MsgType = iota
	MIDI
)

const (
	NOTE_ON NoteState = iota
	NOTE_OFF
)
