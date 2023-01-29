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
	"github.com/rapidmidiex/rmxtui/rtt"
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
		msg wsmsg.MIDIMsg
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

		// Roundtrip timer for messages.
		rtTimer   *rtt.Timer
		pingStats rtt.Stats
		userName  string
		userID    uuid.UUID

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

		rtTimer:   rtt.NewTimer(),
		pingStats: rtt.NewStats(),
		log:       log.Default(),
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

		// Curious to see how long WriteJSON takes
		preSendTime := time.Now()
		err = m.Socket.WriteJSON(envelope)
		if err != nil {
			return rmxerr.ErrMsg{Err: fmt.Errorf("writeJSON: %w", err)}
		}
		return sentMsg{
			id:     envelope.ID,
			sentAt: preSendTime,
		}
	}
}
