package lobbyui

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rapidmidiex/rmxtui/styles"
	"golang.org/x/term"
)

var (
	docStyle = styles.DocStyle
)

// Message types
type ErrMsg struct{ Err error }

type Jam struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	PlayerCount int    `json:"playerCount"`
}

type jamsResp struct {
	Rooms []Jam `json:"rooms"`
}

type jamCreated struct {
	ID string `json:"id"`
}

// For messages that contain errors it's often handy to also implement the
// error interface on the message.
func (e ErrMsg) Error() string { return e.Err.Error() }

// Commands
func (m Model) listJams() tea.Cmd {
	return func() tea.Msg {
		// Create an HTTP client and make a GET request.
		c := &http.Client{Timeout: 10 * time.Second}
		res, err := c.Get(m.apiURL + "/jam")
		if err != nil {
			// There was an error making our request. Wrap the error we received
			// in a message and return it.
			return ErrMsg{err}
		}
		// We received a response from the server.
		// Return the HTTP status code
		// as a message.
		if res.StatusCode >= 400 {
			return ErrMsg{fmt.Errorf("could not get sessions: %d", res.StatusCode)}
		}
		decoder := json.NewDecoder(res.Body)
		var resp jamsResp
		err = decoder.Decode(&resp)
		if err != nil {
			return ErrMsg{Err: fmt.Errorf("decode: %v", err)}
		}
		return resp
	}
}

type Model struct {
	apiURL   string // REST API base endpoint
	jams     []Jam
	jamTable table.Model
	help     tea.Model
	loading  bool
	err      error
	log      log.Logger
}

func New(apiURL string) tea.Model {
	return Model{
		apiURL:  apiURL,
		help:    NewHelpModel(),
		loading: true,
		log:     *log.Default(),
	}
}

// Init is used to handle any initial I/O
func (m Model) Init() tea.Cmd {
	return m.listJams()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.jamTable.SetWidth(msg.Width - 10)
	case ErrMsg:
		// There was an error. Note it in the model.
		m.err = msg
	case jamsResp:
		m.jams = msg.Rooms
		m.jamTable = makeJamsTable(m)
		m.jamTable.Focus()
		m.loading = false
	case jamCreated:
		jamID := msg.ID
		// Auto join the newly created Jam
		cmds = append(cmds, jamSelect(jamID))
	case tea.KeyMsg:
		switch msg.String() {
		case tea.KeyEnter.String():
			jamID := m.jamTable.SelectedRow()[1]

			cmds = append(cmds, jamSelect(jamID))
		case "n":
			// Create new Jam Session
			cmds = append(cmds, jamCreate(m.apiURL))
		}
	}
	newJamTable, jtCmd := m.jamTable.Update(msg)
	m.jamTable = newJamTable

	newHelp, hCmd := m.help.Update(msg)
	m.help = newHelp

	cmds = append(cmds, jtCmd, hCmd)
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	physicalWidth, _, _ := term.GetSize(int(os.Stdout.Fd()))
	doc := strings.Builder{}

	// Jam Session Table
	{
		if len(m.jams) > 0 {
			jamTable := styles.BaseStyle.Width(styles.Width).Render(m.jamTable.View())
			doc.WriteString(jamTable)
		} else if !m.loading {
			doc.WriteString(styles.MessageText.Render("No Jams Yet. Create one?\n\n"))
		}
	}

	// Help menu
	{
		doc.WriteString("\n" + styles.HelpMenu.Render(m.help.View()))
	}

	if physicalWidth > 0 {
		docStyle = styles.DocStyle.MaxWidth(physicalWidth)
	}

	// Okay, let's print it
	return docStyle.Render(doc.String())
}

// https://github.com/rog-golang-buddies/rapidmidiex-research/issues/9#issuecomment-1204853876
func makeJamsTable(m Model) table.Model {
	columns := []table.Column{
		{Title: "Name", Width: 15},
		{Title: "ID", Width: 15},
		{Title: "Players", Width: 10},
		// {Title: "Latency", Width: 4},
	}

	rows := make([]table.Row, 0)

	for _, j := range m.jams {
		row := table.Row{j.Name, j.ID, fmt.Sprintf("%d", j.PlayerCount)}
		rows = append(rows, row)
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return t
}

type JamSelected struct {
	ID string
}

// Commands
func jamSelect(id string) tea.Cmd {
	return func() tea.Msg {
		return JamSelected{id}
	}
}

func jamCreate(baseURL string) tea.Cmd {
	// For now, we're just creating the Jam Session without
	// and options.
	// Next step would be to show inputs for Jam details
	// (name, bpm, etc) before creating the Jam.
	return func() tea.Msg {
		resp, err := http.Post(baseURL+"/jam", "application/json", strings.NewReader("{}"))
		if err != nil {
			return ErrMsg{Err: fmt.Errorf("jamCreate: %v", err)}
		}
		var body jamCreated
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&body)
		if err != nil {
			return ErrMsg{Err: fmt.Errorf("decode: %v", err)}
		}
		return body
	}
}
