package rmxtui

import (
	"fmt"
	"net/url"
	"os"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
	"github.com/hyphengolang/prelude/types/suid"

	"github.com/rapidmidiex/rmxtui/jamui"
	"github.com/rapidmidiex/rmxtui/keymap"
	"github.com/rapidmidiex/rmxtui/lobbyui"
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
		curView      appView
		lobby        tea.Model
		jam          tea.Model
		RESTendpoint string
		WSendpoint   string
		// jamSocket    *websocket.Conn // Websocket connection to a Jam Session
	}
)

const (
	jamView appView = iota
	lobbyView
)

func NewModel(serverHostURL string) (mainModel, error) {
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
		m.curError = msg.err.Error()

		// Was a key press
	case tea.KeyMsg:
		switch {
		// Ctrl+c exits. Even with short running programs it's good to have
		// a quit key, just incase your logic is off. Users will be very
		// annoyed if they can't exit.
		case key.Matches(msg, keymap.DefaultMapping.Quit):
			return m, tea.Quit
		}

	case lobbyui.JamSelected:
		cmd = m.jamConnect(msg.ID)
		m.curView = jamView
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
	serverLine := fmt.Sprintf("\nServer: %s\n", m.RESTendpoint)

	switch m.curView {
	case jamView:
		return serverLine + m.jam.View()
	default:
		return serverLine + m.lobby.View()
	}
}

func Run(serverHostURL string) {
	m, err := NewModel(serverHostURL)
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
