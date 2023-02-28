// Package wsmmsg contains the RMX message types for communication between clients.
package wsmsg

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type (
	MsgType   int
	NoteState int

	Envelope struct {
		// Message identifier
		ID uuid.UUID `json:"id"`
		// TextMsg | MIDIMsg | ConnectMsg
		Typ MsgType `json:"type"`
		// RMX client identifier
		UserID uuid.UUID `json:"userId"`
		// Actual message data.
		Payload json.RawMessage `json:"payload"`
	}

	TextMsg struct {
		DisplayName string `json:"displayName"`
		Body        string `json:"body"`
	}

	MIDIMsg struct {
		State NoteState `json:"state"`
		// MIDI Note # in "C3 Convention", C3 = 60. Available values: (0-127)
		Number int `json:"number"`
		// MIDI Velocity (0-127)
		Velocity int `json:"velocity"`
	}

	ConnectMsg struct {
		UserID   uuid.UUID `json:"userId"`
		UserName string    `json:"userName"`
	}
)

const (
	TEXT MsgType = iota
	MIDI
	CONNECT
)

const (
	NOTE_OFF NoteState = iota
	NOTE_ON
)

func (e *Envelope) SetPayload(payload any) error {
	p, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	e.Payload = p
	return nil
}

func (e *Envelope) Unwrap(msg any) error {
	return json.Unmarshal(e.Payload, msg)
}

func (t *MsgType) UnmarshalJSON(data []byte) error {
	var rawType string
	err := json.Unmarshal(data, &rawType)
	if err != nil {
		return err
	}

	switch rawType {
	case "connect":
		*t = CONNECT
	case "midi":
		*t = MIDI
	case "text":
		*t = TEXT
	default:
		return fmt.Errorf("unknown type: %s", rawType)
	}
	return nil
}

func (t *MsgType) MarshalJSON() ([]byte, error) {
	switch *t {
	case CONNECT:
		return []byte(`"connect"`), nil
	case MIDI:
		return []byte(`"midi"`), nil
	case TEXT:
		return []byte(`"text"`), nil
	}
	return []byte{}, fmt.Errorf("unknown MsgTyp value: %d", *t)
}
