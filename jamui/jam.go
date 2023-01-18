package jamui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gorilla/websocket"
	"github.com/rapidmidiex/rmxtui/keymap"
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

// Command Types
type Connected struct {
	WS    *websocket.Conn
	JamID string
}

type LeaveRoom struct {
	Err error
}

type pianoKey struct {
	noteNumber int    // MIDI note number ie: 72
	name       string // Name of musical note, ie: "C5"
	keyMap     string // Mapped qwerty keyboard key. Ex: "q"
}

type Model struct {
	// Piano keys. {"q": pianoKey{72, "C5", "q", ...}}
	piano []pianoKey
	// Currently active piano keys
	activeKeys map[string]struct{}
	// Websocket connection for current Jam Session
	Socket *websocket.Conn
	// Jam Session ID
	ID string
}

func New() Model {
	return Model{
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
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
	// Commands go here
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keymap.DefaultMapping.GoBack):
			cmd = m.leaveRoom()
			cmds = append(cmds, cmd)
		default:
			fmt.Printf("Key press: %s\n", msg.String())
			return m, nil
		}

	// Entered the Jam Session
	case Connected:
		m.Socket = msg.WS
		m.ID = msg.JamID
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
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
	doc.WriteString(keyboard + "\n\n")
	return docStyle.Render(doc.String())
}

// LeaveRoom disconnects from the room and sends a LeaveRoom message.
func (m Model) leaveRoom() tea.Cmd {
	return func() tea.Msg {
		err := m.Socket.Close()
		return LeaveRoom{Err: err}
	}
}
