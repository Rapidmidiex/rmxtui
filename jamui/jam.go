package jamui

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rapidmidiex/rmxtui/chatui"
	"github.com/rapidmidiex/rmxtui/keymap"
	"github.com/rapidmidiex/rmxtui/midi"
	"github.com/rapidmidiex/rmxtui/rmxerr"
	"github.com/rapidmidiex/rmxtui/rtt"
	"github.com/rapidmidiex/rmxtui/vpiano"
	"github.com/rapidmidiex/rmxtui/wsmsg"
	"golang.org/x/term"
)

// DocStyle styling for viewports
var (
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	docStyle = lipgloss.NewStyle().Padding(1, 2, 1, 2)

	keyBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "-",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "╰",
		BottomRight: "╯",
	}

	pianoKeyStyle = lipgloss.NewStyle().
			Align(lipgloss.Center).
			Border(keyBorder, true).
			BorderForeground(highlight).
			Padding(0, 1)
)

const (
	chatFocus focused = iota
	pianoFocus
	// Don't forget to update model.availableFocusStates if more states are added here.
)

type (
	// Command Types
	ConnectedMsg struct {
		WS    *websocket.Conn
		JamID string
	}

	LeaveRoomMsg struct{}

	StatsMsg rtt.Stats

	sentMsg struct {
		id     uuid.UUID
		sentAt time.Time
	}

	recvConnectMsg struct {
		userID   uuid.UUID
		userName string
	}

	recvMIDIMsg struct {
		id  uuid.UUID
		msg wsmsg.MIDIMsg
	}

	focused int

	model struct {
		// Piano keys.
		pianoNotes vpiano.Notes
		// Currently active piano keys
		activeKeys map[string]struct{}

		// Websocket client
		wsClient *wsClient

		// Jam Session ID
		ID string
		// Chat container
		chatBox tea.Model

		// Element currently with focus
		focused focused
		// Number of available focus status
		availableFocusStates int

		// Roundtrip timer for messages.
		rtTimer   *rtt.Timer
		pingStats rtt.Stats
		userName  string
		userID    uuid.UUID

		curMidiMsg  wsmsg.MIDIMsg
		midiPlayer  midi.Synth
		audioPlayer *audioPlayer
		sampleRate  beep.SampleRate
		noteKeyMap  vpiano.NoteKeyMap
		log         *log.Logger
	}

	wsClient struct {
		mu sync.Mutex
		// Websocket connection for current Jam Session
		conn *websocket.Conn
	}

	audioPlayer struct {
		mu    sync.Mutex
		mixer *beep.Mixer
	}
)

func New() (model, error) {
	midiPlayer, err := midi.NewSynth(midi.NewSynthOpts{
		// TODO: Take soundFont as arg, or select in TUI
		SoundFontName: midi.GeneralUser,
	})
	if err != nil {
		return model{}, fmt.Errorf("midi.NewPlayer: %w", err)
	}

	sr := beep.SampleRate(44100)
	// TODO: Determine buffer length sweet spot.
	// Bigger -> less CPU, slower response
	// Lower -> more CPU, faster response
	bufLen := sr.N(time.Millisecond * 20)
	err = speaker.Init(sr, bufLen)
	if err != nil {
		return model{}, fmt.Errorf("speaker.Init: %w", err)
	}

	pianoNotes := vpiano.MakeOctaveNotes(vpiano.C4)

	m := model{
		pianoNotes: pianoNotes,
		activeKeys: make(map[string]struct{}),

		chatBox: chatui.New(),

		focused: chatFocus,
		// If more focus states are added, update number of available states
		availableFocusStates: 2,

		rtTimer:     rtt.NewTimer(),
		pingStats:   rtt.NewStats(),
		noteKeyMap:  pianoNotes.ToBindingMap(),
		midiPlayer:  midiPlayer,
		audioPlayer: &audioPlayer{mixer: &beep.Mixer{}},
		sampleRate:  sr,
		log:         log.Default(),
	}

	speaker.Play(m.audioPlayer.mixer)
	return m, nil
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.chatBox.Init(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keymap.DefaultMapping.Quit):
			cmds = append(cmds, m.leaveRoom())
		case key.Matches(msg, keymap.DefaultMapping.GoBack):
			cmds = append(cmds, m.leaveRoom())

		case key.Matches(msg, keymap.DefaultMapping.CycleFocus):
			// Keep the state in bounds of the number of available states
			m.focused = (m.focused + 1) % focused(m.availableFocusStates)
			m.chatBox, cmd = m.chatBox.Update(chatui.ToggleFocusMsg{})
			cmds = append(cmds, cmd)
		}

		switch m.focused {
		case chatFocus:
			m.chatBox, cmd = m.chatBox.Update(msg)
			cmds = append(cmds, cmd)
		case pianoFocus:
			// TODO: highlight the key play
			cmds = append(cmds, m.sendMIDIMessage(msg.String()))
		}
		// *** End KeyMsg ***
		return m, tea.Batch(cmds...)

	// Entered the Jam Session
	case ConnectedMsg:
		m.wsClient = &wsClient{conn: msg.WS}
		m.ID = msg.JamID
		cmds = append(cmds, m.listenSocket())

	case chatui.SendMsg:
		cmds = append(cmds, m.sendTextMessage(msg.Msg))
	case sentMsg:
		// TODO Delete me after testing vv
		// Curious if this time includes latency.
		// If this number is 0, delete this log
		timeToSend := time.Since(msg.sentAt).Milliseconds()
		if timeToSend > 0 {
			m.log.Printf("Send time: %d (ms)", timeToSend)
		}
		// TODO Delete me after testing ^^

		// Timer started after message sent.
		// Does not include time to send message.
		err := m.rtTimer.Start(msg.id.String())
		cmd := func() tea.Msg { return rmxerr.ErrMsg{Err: err} }
		cmds = append(cmds, cmd)

	case chatui.RecvTextMsg:
		m.chatBox, cmd = m.chatBox.Update(msg)

		// TODO: Move to envelope msg handler (not just text)
		latest := m.rtTimer.Stop(msg.ID.String())
		m.pingStats = m.pingStats.Calc(latest)
		pingCmd := func() tea.Msg { return StatsMsg(m.pingStats) }

		// Start listening again
		cmds = append(cmds, cmd, m.listenSocket(), pingCmd)

	case recvConnectMsg:
		m.userName = msg.userName
		m.userID = msg.userID
		// Start listening again
		cmds = append(cmds, cmd, m.listenSocket())
	case recvMIDIMsg:
		m.curMidiMsg = msg.msg

		// TODO: Move to envelope msg handler (not just text)
		latest := m.rtTimer.Stop(msg.id.String())
		m.pingStats = m.pingStats.Calc(latest)
		pingCmd := func() tea.Msg { return StatsMsg(m.pingStats) }

		// Play MIDI on speakers
		cmd = m.playMIDI(msg.msg)
		// Start listening again
		cmds = append(cmds, cmd, m.listenSocket(), pingCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	physicalWidth, _, _ := term.GetSize(int(os.Stdout.Fd()))
	doc := strings.Builder{}

	if physicalWidth > 0 {
		docStyle = docStyle.MaxWidth(physicalWidth)
	}

	doc.WriteString(m.chatBox.View())
	doc.WriteString(m.renderPiano() + "\n\n")
	return docStyle.Render(doc.String())
}

// LeaveRoom disconnects from the room and sends a LeaveRoom message.
func (m model) leaveRoom() tea.Cmd {
	return func() tea.Msg {
		if m.wsClient.conn == nil {
			return LeaveRoomMsg{}
		}
		// Send websocket close message
		err := m.wsClient.conn.WriteControl(
			websocket.CloseMessage,
			nil,
			time.Now().Add(time.Second*10),
		)
		if err != nil {
			return rmxerr.ErrMsg{Err: err}
		}
		return LeaveRoomMsg{}
	}
}

// ListenSocket reads messages from the websocket connection and returns a chatui.RecvMsg.
func (m model) listenSocket() tea.Cmd {
	// https://github.com/charmbracelet/bubbletea/issues/25#issuecomment-732339162
	return func() tea.Msg {
		var message wsmsg.Envelope
		err := m.wsClient.readMsg(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				return rmxerr.ErrMsg{Err: fmt.Errorf("readJSON: unexpected close: %w", err)}
			}
			return rmxerr.ErrMsg{Err: fmt.Errorf("readJSON: %w", err)}
		}

		switch message.Typ {
		case wsmsg.TEXT:
			var textMsg wsmsg.TextMsg
			if err := message.Unwrap(&textMsg); err != nil {
				return rmxerr.ErrMsg{Err: fmt.Errorf("unmarshal TextMsg: %+v\n%w", message, err)}
			}
			fromSelf := false
			if message.UserID.String() == m.userID.String() {
				fromSelf = true
			}
			return chatui.RecvTextMsg{
				ID:          message.ID,
				DisplayName: textMsg.DisplayName,
				Msg:         string(textMsg.Body),
				FromSelf:    fromSelf,
			}

		case wsmsg.CONNECT:
			var conMsg wsmsg.ConnectMsg
			if err := message.Unwrap(&conMsg); err != nil {
				return rmxerr.ErrMsg{Err: fmt.Errorf("unmarshal ConnectMsg: %+v\n%w", message, err)}
			}
			return recvConnectMsg{
				userName: conMsg.UserName,
				userID:   conMsg.UserID,
			}

		case wsmsg.MIDI:
			var midiMsg wsmsg.MIDIMsg
			if err := message.Unwrap(&midiMsg); err != nil {
				return rmxerr.ErrMsg{Err: fmt.Errorf("unmarshal MIDIMsg: %+v\n%w", message, err)}
			}
			m.log.Printf("MIDI received: %+v\n", midiMsg)
			return recvMIDIMsg{
				id:  message.ID,
				msg: midiMsg,
			}
		default:
			return rmxerr.ErrMsg{Err: fmt.Errorf("unknown message type: %+v", message)}
		}
	}

}

func (m model) sendTextMessage(body string) tea.Cmd {
	return func() tea.Msg {
		envelope := wsmsg.Envelope{
			ID:     uuid.New(),
			Typ:    wsmsg.TEXT,
			UserID: m.userID,
		}
		textMsg := wsmsg.TextMsg{Body: body, DisplayName: m.userName}
		err := envelope.SetPayload(textMsg)
		if err != nil {
			return rmxerr.ErrMsg{Err: fmt.Errorf("marshal: %w", err)}
		}

		// Curious to see how long WriteJSON takes
		preSendTime := time.Now()
		err = m.wsClient.conn.WriteJSON(envelope)
		if err != nil {
			return rmxerr.ErrMsg{Err: fmt.Errorf("writeJSON: %w", err)}
		}
		return sentMsg{
			id:     envelope.ID,
			sentAt: preSendTime,
		}
	}
}

func (m model) sendMIDIMessage(keyPressed string) tea.Cmd {
	return func() tea.Msg {
		midiNum := m.noteKeyMap[keyPressed].MIDI
		if !vpiano.InRange(midiNum) {
			return nil
		}

		msg := wsmsg.MIDIMsg{
			State:    wsmsg.NOTE_ON,
			Velocity: 127,
			Number:   midiNum,
		}

		envelope := wsmsg.Envelope{
			ID:     uuid.New(),
			Typ:    wsmsg.MIDI,
			UserID: m.userID,
		}

		if err := envelope.SetPayload(msg); err != nil {
			return rmxerr.ErrMsg{Err: fmt.Errorf("marshal: %w", err)}
		}
		if err := m.wsClient.writeMsg(envelope); err != nil {
			return rmxerr.ErrMsg{Err: fmt.Errorf("writeJSON: %w", err)}
		}
		return sentMsg{
			id:     envelope.ID,
			sentAt: time.Now(),
		}
	}
}

// PlayMIDI plays the given MIDI note through system audio.
func (m *model) playMIDI(note wsmsg.MIDIMsg) tea.Cmd {
	return func() tea.Msg {
		// NOTE_OFF messages are not really going to work with a virtual keyboard
		// or with sending realtime messages, so we have to use some arbitrary duration to play the note.
		// TODO: Maybe control duration with some other key
		duration := time.Second * 2
		s := midi.NewMIDIStreamer(duration)

		// Render MIDI note to audio streamer buffer
		if err := m.midiPlayer.Render(note, s); err != nil {
			return rmxerr.ErrMsg{Err: err}
		}

		// Take n seconds worth of samples @ 44.1khz from the audio streamer and
		// add it to the main speaker mix.
		m.audioPlayer.addToMix(beep.Take(m.sampleRate.N(duration), s))

		return nil
	}
}

func (m model) renderPiano() string {
	pianoKeys := make([]string, 0)
	for _, v := range m.pianoNotes {
		if v.IsAccidental {
			// TODO: Figure out black keys
			continue
		}
		pianoKeys = append(pianoKeys,
			pianoKeyStyle.Render(lipgloss.JoinVertical(lipgloss.Top, v.Name, "\n", fmt.Sprintf("(%s)", v.KeyBinding))),
		)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, pianoKeys...)
}

func (c *wsClient) readMsg(out *wsmsg.Envelope) error {
	return c.conn.ReadJSON(out)
}

func (c *wsClient) writeMsg(envelope wsmsg.Envelope) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(envelope)
}

func (a *audioPlayer) addToMix(s beep.Streamer) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.mixer.Add(s)
}
