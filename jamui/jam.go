package jamui

import (
	"fmt"
	"log"
	"os"
	"strings"

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

		conn chan wsmsg.TextMsg

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
			m.chatBox, cmd = m.chatBox.Update(chatui.ToggleFocus{})
			cmds = append(cmds, cmd)
		}

	// Entered the Jam Session
	case ConnectedMsg:
		m.Socket = msg.WS
		m.ID = msg.JamID
		m.conn = make(chan wsmsg.TextMsg)
		cmds = append(cmds, m.readPump(), socketListen(m.conn))

	case chatui.SendMsg:
		err := m.Socket.WriteJSON(wsmsg.TextMsg{Typ: wsmsg.TEXT, Payload: msg.Msg})
		if err != nil {
			// TODO bubble error up
			fmt.Printf("ERROR: %v", err)
		}
		cmds = append(cmds, socketListen(m.conn))
	case chatui.RecvMsg:
		m.chatBox, cmd = m.chatBox.Update(msg)
		cmds = append(cmds, cmd, socketListen(m.conn))
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
		err := m.Socket.Close()
		return LeaveRoom{Err: err}
	}
}

func socketListen(c <-chan wsmsg.TextMsg) tea.Cmd {
	// https://github.com/charmbracelet/bubbletea/issues/25
	// TODO: I think this command is not called enough to read all messages
	// from the channel. When the readPump writes to the channel, It does not send a Msg, so bubbletea doesnt know to read again.
	// I'll have to think about how to continually read incoming messages.

	// Repro:
	// Send > 3 messages from another client.
	// You see all messages are read from the socket in the readPump() loop, but only the first few messages are passed to bubbletea through this RecvMsg here.
	// socketListen needs to be invoked more often (in a loop?)
	return func() tea.Msg {
		txtMsg := <-c
		return chatui.RecvMsg{
			Msg: txtMsg.Payload,
		}
	}
}

func (m model) readPump() tea.Cmd {
	return func() tea.Msg {
		defer func() {
			m.log.Print("CLOSE")
			m.Socket.Close()
		}()
		for {
			m.log.Print("LISTEN")
			var message wsmsg.TextMsg
			err := m.Socket.ReadJSON(&message)
			m.log.Printf("MESSAGE: %s", message.Payload)
			if err != nil {
				m.log.Printf("ERROR: %s", err)
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("error: %v", err)
				}
				break
			}
			m.conn <- message
		}
		return nil
	}
}
