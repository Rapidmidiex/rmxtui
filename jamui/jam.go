package jamui

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rapidmidiex/rmxtui/chatui"
	"github.com/rapidmidiex/rmxtui/keymap"
	"github.com/rapidmidiex/rmxtui/rmxerr"
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

	sentMsg struct {
		id     uuid.UUID
		sentAt time.Time
	}

	recvConnectMsg struct {
		userID   uuid.UUID
		userName string
	}

	recvMIDIMsg struct {
		msg wsmsg.MIDIMsg
	}

	PingCalcMsg struct {
		Latest time.Duration
		Avg    time.Duration
		Min    time.Duration
		Max    time.Duration
	}

	// Virtual keyboard types
	pianoKey struct {
		noteNumber int    // MIDI note number ie: 72
		name       string // Name of musical note, ie: "C5"
		keyMap     string // Mapped qwerty keyboard key. Ex: "q"
	}

	focused int

	model struct {
		// Piano keys. {"q": pianoKey{72, "C5", "q", ...}}
		piano []pianoKey
		// Currently active piano keys
		activeKeys map[string]struct{}
		// Websocket connection for current Jam Session
		Socket *websocket.Conn
		// Jam Session ID
		ID string
		// Chat container
		chatBox tea.Model

		// Element currently with focus
		focused focused
		// Number of available focus status
		availableFocusStates int

		// Map of times the latest messages were sent.
		// { [messageID]: timeSentAt }
		lastMsgs map[string]time.Time

		// List of latest roundtrip times for messages.
		pings    []time.Duration
		userName string
		userID   uuid.UUID

		curMidiMsg wsmsg.MIDIMsg

		log *log.Logger
	}
)

func New() model {
	return model{
		piano: []pianoKey{
			{72, "C5", "q"},
			{74, "D5", "w"},
			{76, "E5", "e"},
			{77, "F5", "r"},
			{79, "G5", "t"},
			{81, "A5", "y"},
			{83, "B5", "u"},
			{84, "C6", "i"},
		},

		activeKeys: make(map[string]struct{}),

		chatBox: chatui.New(),

		focused: chatFocus,
		// If more focus states are added, update number of available states
		availableFocusStates: 2,

		lastMsgs: make(map[string]time.Time),
		pings:    make([]time.Duration, 0),

		log: log.Default(),
	}
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
			// TODO: play the piano
			// - highlight the key play
			// - run send note command
		}
		// *** End KeyMsg ***
		return m, tea.Batch(cmds...)

	// Entered the Jam Session
	case ConnectedMsg:
		m.Socket = msg.WS
		m.ID = msg.JamID
		cmds = append(cmds, m.listenSocket())

	case chatui.SendMsg:
		cmds = append(cmds, m.sendTextMessage(msg.Msg, m.userName))
	case sentMsg:
		m.lastMsgs[msg.id.String()] = msg.sentAt

	case chatui.RecvTextMsg:
		m.chatBox, cmd = m.chatBox.Update(msg)

		ping := m.calcPing(msg.ID.String())
		if ping > 0 {
			m.pings = append(m.pings, ping)
		}
		// Start listening again
		cmds = append(cmds, cmd, m.listenSocket(), updatePing(ping))

	case recvConnectMsg:
		m.userName = msg.userName
		m.userID = msg.userID
		// Start listening again
		cmds = append(cmds, cmd, m.listenSocket())
	case recvMIDIMsg:
		m.curMidiMsg = msg.msg
		// Start listening again
		cmds = append(cmds, cmd, m.listenSocket())
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	physicalWidth, _, _ := term.GetSize(int(os.Stdout.Fd()))
	doc := strings.Builder{}

	if physicalWidth > 0 {
		docStyle = docStyle.MaxWidth(physicalWidth)
	}

	// Keyboard
	keyboard := lipgloss.JoinHorizontal(lipgloss.Top,
		pianoKeyStyle.Render("C5"+"\n\n"+"(q)"),
		pianoKeyStyle.Render("D5"+"\n\n"+"(w)"),
		pianoKeyStyle.Render("E5"+"\n\n"+"(e)"),
		pianoKeyStyle.Render("F5"+"\n\n"+"(r)"),
		pianoKeyStyle.Render("G5"+"\n\n"+"(t)"),
		pianoKeyStyle.Render("A5"+"\n\n"+"(y)"),
		pianoKeyStyle.Render("B5"+"\n\n"+"(u)"),
		pianoKeyStyle.Render("C6"+"\n\n"+"(i)"),
	)
	doc.WriteString(m.chatBox.View())
	doc.WriteString(keyboard + "\n\n")
	return docStyle.Render(doc.String())
}

// LeaveRoom disconnects from the room and sends a LeaveRoom message.
func (m model) leaveRoom() tea.Cmd {
	return func() tea.Msg {
		if m.Socket == nil {
			return LeaveRoomMsg{}
		}
		// Send websocket close message
		err := m.Socket.WriteControl(
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
		err := m.Socket.ReadJSON(&message)
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
			m.log.Println(midiMsg)
			return recvMIDIMsg{
				msg: midiMsg,
			}
		default:
			return rmxerr.ErrMsg{Err: fmt.Errorf("unknown message type: %+v", message)}
		}
	}

}

func (m model) sendTextMessage(body, displayName string) tea.Cmd {
	return func() tea.Msg {
		envelope := wsmsg.Envelope{
			ID:     uuid.New(),
			Typ:    wsmsg.TEXT,
			UserID: m.userID,
		}
		textMsg := wsmsg.TextMsg{Body: body, DisplayName: displayName}
		err := envelope.SetPayload(textMsg)
		if err != nil {
			return rmxerr.ErrMsg{Err: fmt.Errorf("marshal: %w", err)}
		}
		err = m.Socket.WriteJSON(envelope)
		if err != nil {
			return rmxerr.ErrMsg{Err: fmt.Errorf("writeJSON: %w", err)}
		}
		return sentMsg{
			id:     envelope.ID,
			sentAt: time.Now(),
		}
	}
}

// CalcPing looks up the message in the message history and calculates the roundtrip time.
// If the message is not found, -1 is returned.
func (m model) calcPing(msgID string) time.Duration {
	sentAt, ok := m.lastMsgs[msgID]
	if !ok {
		return -1
	}

	delete(m.lastMsgs, msgID)
	return time.Since(sentAt)
}

func updatePing(ping time.Duration) tea.Cmd {
	return func() tea.Msg {
		return PingCalcMsg{
			Latest: ping,
			// TODO:
			Avg: 0,
			Max: 0,
			Min: 0,
		}
	}
}
