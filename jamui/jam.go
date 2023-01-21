package jamui

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gorilla/websocket"
	"github.com/rapidmidiex/rmxtui/chatui"
	"github.com/rapidmidiex/rmxtui/keymap"
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

	LeaveRoom struct {
		Err error
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

		// Info about the last websocket message sent.
		lastMsg struct {
			id string
			// Time last websocket message was sent.
			sentAt time.Time
			// Difference between lastMsg.sentAt and when this message was received from server broadcast.
			ping time.Duration
		}

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
		case key.Matches(msg, keymap.DefaultMapping.GoBack):
			cmd = m.leaveRoom()
			cmds = append(cmds, cmd)

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
		m.lastMsg.sentAt = time.Now()
		m.lastMsg.id = msg.Msg
		m.lastMsg.ping = 1<<63 - 1 // Max duration (reset)

		err := m.Socket.WriteJSON(wsmsg.TextMsg{Typ: wsmsg.TEXT, Payload: msg.Msg})
		if err != nil {
			// TODO bubble error up
			m.log.Printf("ERROR: %v", err)
		}
	case chatui.RecvMsg:
		m.chatBox, cmd = m.chatBox.Update(msg)
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
		// Send websocket close message
		err := m.Socket.WriteControl(
			websocket.CloseMessage,
			nil,
			time.Now().Add(time.Second*10),
		)
		return LeaveRoom{Err: err}
	}
}

// ListenSocket reads messages from the websocket connection and returns a chatui.RecvMsg.
func (m model) listenSocket() tea.Cmd {
	// https://github.com/charmbracelet/bubbletea/issues/25#issuecomment-732339162
	return func() tea.Msg {
		var message wsmsg.TextMsg
		err := m.Socket.ReadJSON(&message)
		if err != nil {
			m.log.Printf("ERROR: %s", err)
			// TODO: Handle error
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				m.log.Printf("error: %v", err)
			}
			// TODO return errMsg
			return nil
		}

		return chatui.RecvMsg{
			Msg: message.Payload,
		}
	}

}
