package rmxtui

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gorilla/websocket"
	"github.com/hyphengolang/prelude/types/suid"
	"golang.org/x/term"

	"github.com/rapidmidiex/rmxtui/jamui"
	"github.com/rapidmidiex/rmxtui/keymap"
	"github.com/rapidmidiex/rmxtui/lobbyui"
	"github.com/rapidmidiex/rmxtui/styles"
)

// ********
// Code heavily based on "Project Journal"
// https://github.com/bashbunni/pjs
// https://www.youtube.com/watch?v=uJ2egAkSkjg&t=319s
// ********

type (
	Session struct {
		Id suid.UUID `json:"id"`
		// UserCount int    `json:"userCount"`
	}

	appView int

	// Message types
	errMsg struct{ err error }

	mainModel struct {
		loading      bool
		curError     error
		curView      appView
		lobby        tea.Model
		jam          tea.Model
		RESTendpoint string
		WSendpoint   string
		// jamSocket    *websocket.Conn // Websocket connection to a Jam Session
		log log.Logger
	}
)

const (
	jamView appView = iota
	lobbyView
)

var (
	docStyle = styles.DocStyle
)

func NewModel(serverHostURL string, debugMode bool) (mainModel, error) {
	wsHostURL, err := url.Parse(serverHostURL)
	if err != nil {
		return mainModel{}, err
	}
	wsHostURL.Scheme = "ws"

	return mainModel{
		curView:      lobbyView,
		lobby:        lobbyui.New(serverHostURL + "/api/v1"),
		jam:          jamui.New(),
		RESTendpoint: serverHostURL + "/api/v1",
		WSendpoint:   wsHostURL.String() + "/ws",
		log:          *log.Default(),
	}, nil
}

func (m mainModel) Init() tea.Cmd {
	return tea.Batch(
		m.lobby.Init(),
		m.jam.Init(),
	)
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	// Handle incoming messages from I/O
	switch msg := msg.(type) {
	case errMsg:
		m.curError = msg.err
	case lobbyui.ErrMsg:
		m.curError = msg.Err

		// Was a key press
	case tea.KeyMsg:
		switch {
		// Ctrl+c exits. Even with short running programs it's good to have
		// a quit key, just incase your logic is off. Users will be very
		// annoyed if they can't exit.
		case key.Matches(msg, keymap.DefaultMapping.Quit):
			return m, tea.Quit
		}

	case jamui.Connected:
		m.curView = jamView
	case lobbyui.JamSelected:
		cmd = m.jamConnect(msg.ID)
		cmds = append(cmds, cmd)
	case jamui.LeaveRoom:
		if msg.Err != nil {
			cmd = m.handleError(fmt.Errorf("leaveRoom: %w", msg.Err))
			cmds = append(cmds, cmd)
		}
		m.curView = lobbyView
	}

	// Call sub-model Updates
	switch m.curView {
	case lobbyView:
		m.lobby, cmd = m.lobby.Update(msg)
	case jamView:
		m.jam, cmd = m.jam.Update(msg)
	}

	// Run all commands from sub-model Updates
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)

}

func (m mainModel) View() string {
	physicalWidth, _, _ := term.GetSize(int(os.Stdout.Fd()))
	doc := strings.Builder{}
	status := fmt.Sprintf("server: %s", m.RESTendpoint)

	if m.loading {
		status = "Fetching Jam Sessions..."
	}

	if m.curError != nil {
		status = styles.RenderError(fmt.Sprint(m.curError))
	}

	switch m.curView {
	case jamView:
		doc.WriteString("\n" + m.jam.View())

	case lobbyView:
		doc.WriteString("\n" + m.lobby.View())
	}

	// Status bar
	{
		w := lipgloss.Width

		statusKey := styles.StatusStyle.Render("STATUS")
		statusVal := styles.StatusText.Copy().
			Width(styles.Width - w(statusKey)).
			Render(status)

		bar := lipgloss.JoinHorizontal(lipgloss.Top,
			statusKey,
			statusVal,
		)

		if physicalWidth > 0 {
			docStyle = styles.DocStyle.MaxWidth(physicalWidth)
		}
		doc.WriteString("\n" + styles.StatusBarStyle.Width(styles.Width).Render(bar))
	}
	return docStyle.Render(doc.String())
}

func Run(serverHostURL string, debugMode bool) {
	m, err := NewModel(serverHostURL, debugMode)
	if err != nil {
		bail(err)
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		bail(err)
	}
}

func bail(err error) {
	if err != nil {
		fmt.Printf("Uh oh, there was an error: %v\n", err)
		os.Exit(1)
	}
}

func (m mainModel) jamConnect(jamID string) tea.Cmd {
	return func() tea.Msg {
		jURL := m.WSendpoint + "/jam/" + jamID
		ws, _, err := websocket.DefaultDialer.Dial(jURL, nil)
		if err != nil {
			return errMsg{fmt.Errorf("jamConnect: %v\n%v", jURL, err)}
		}
		return jamui.Connected{
			WS:    ws,
			JamID: jamID,
		}
	}
}

func (m mainModel) handleError(err error) tea.Cmd {
	return func() tea.Msg {
		return errMsg{err: err}
	}
}
