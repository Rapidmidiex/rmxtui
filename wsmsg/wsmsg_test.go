package wsmsg_test

import (
	"encoding/json"
	"testing"

	"github.com/rapidmidiex/rmxtui/wsmsg"
	"github.com/stretchr/testify/require"
)

func TestMsgTypeMarshaling(t *testing.T) {
	t.Run("unmarshals type from JSON", func(t *testing.T) {
		message := []byte(`{
    "id": "7b0f33ba-8a50-446d-aaa4-4de4aa96fc6c",
    "type": "midi",
    "payload": {
        "state": 1,
        "number": 60,
        "velocity": 120
    },
    "userId": null
}`)

		var got wsmsg.Envelope
		err := json.Unmarshal(message, &got)
		require.NoError(t, err)

		require.Equal(t, got.Typ, wsmsg.MIDI)
	})

	t.Run("marshals type to JSON", func(t *testing.T) {
		message := wsmsg.Envelope{
			Typ: wsmsg.MIDI,
		}

		got, err := json.Marshal(&message)
		require.NoError(t, err)
		want := `"type":"midi"`
		require.Containsf(t, string(got), want, "JSON does not contain [ %s ]\n%s", want, string(got))
	})
}
