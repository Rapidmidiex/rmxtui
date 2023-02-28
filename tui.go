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
	"github.com/rapidmidiex/rmxtui/rmxerr"
	"github.com/rapidmidiex/rmxtui/rtt"
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
	mainModel struct {
		loading      bool
		curError     error
		curView      appView
		lobby        tea.Model
		jam          tea.Model
		RESTendpoint string
		WSendpoint   string
		rttStats     rtt.Stats
		log          log.Logger
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
	jamModel, err := jamui.New()
	if err != nil {
		return mainModel{}, err
	}
	restEndpoint := serverHostURL + "/v0"
	return mainModel{
		curView:      lobbyView,
		lobby:        lobbyui.New(serverHostURL + "/v0"),
		jam:          jamModel,
		RESTendpoint: restEndpoint,
		WSendpoint:   strings.Replace(restEndpoint, "http", "ws", 1),
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
	case rmxerr.ErrMsg:
		m.curError = msg.Err

	case jamui.StatsMsg:
		m.rttStats = rtt.Stats(msg)

		// Was a key press
	case tea.KeyMsg:
		switch {
		// Ctrl+c exits. Even with short running programs it's good to have
		// a quit key, just incase your logic is off. Users will be very
		// annoyed if they can't exit.
		case key.Matches(msg, keymap.DefaultMapping.Quit):
			return m, tea.Quit
		}

	case jamui.ConnectedMsg:
		m.curView = jamView
	case lobbyui.JamSelected:
		cmd = m.jamConnect(msg.ID)
		cmds = append(cmds, cmd)
	case jamui.LeaveRoomMsg:
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
	serverLine := fmt.Sprintf("server: %s", formatHost(m.RESTendpoint))
	status := serverLine
	statusKeyText := "STATUS"

	rttStats := "--"
	if m.rttStats.Min > 0 {
		rttStats = lipgloss.JoinHorizontal(lipgloss.Right,
			"Ping (ms)·",
			fmt.Sprintf("cur %d·", m.rttStats.Latest.Milliseconds()),
			fmt.Sprintf("max %d·", m.rttStats.Max.Milliseconds()),
			fmt.Sprintf("min %d·", m.rttStats.Min.Milliseconds()),
			fmt.Sprintf("avg %d", m.rttStats.Avg.Milliseconds()),
		)
	}

	if m.loading {
		status = "Fetching Jam Sessions..."
		statusKeyText = "LOADING"
	}

	if m.curError != nil {
		status = styles.RenderError(lipgloss.JoinVertical(lipgloss.Top, fmt.Sprint(m.curError), serverLine))
		statusKeyText = "ERROR"
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

		statusKey := styles.StatusStyle.Render(statusKeyText)
		ping := styles.PingStyle.Render(rttStats)
		statusVal := styles.StatusText.Copy().
			Width(styles.Width - w(statusKey) - w(ping)).
			Render(status)
		bar := lipgloss.JoinHorizontal(lipgloss.Right,
			statusKey,
			statusVal,
			ping,
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
		jURL := fmt.Sprintf("%s/jams/%s/ws", m.WSendpoint, jamID)
		ws, _, err := websocket.DefaultDialer.Dial(jURL, nil)
		if err != nil {
			return rmxerr.ErrMsg{Err: fmt.Errorf("jamConnect: %v\n%v", jURL, err)}
		}
		return jamui.ConnectedMsg{
			WS:    ws,
			JamID: jamID,
		}
	}
}

func formatHost(endpoint string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		log.Fatal(err)
	}
	return parsed.Host
}
