package chatui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/rapidmidiex/rmxtui/rmxerr"
)

// Reference:
// https://github.com/charmbracelet/bubbletea/blob/master/examples/chat/main.go

type (
	ToggleFocusMsg struct{}

	SendMsg struct {
		Msg string
	}
	RecvTextMsg struct {
		ID          uuid.UUID
		DisplayName string
		Msg         string
		FromSelf    bool
	}
)

type model struct {
	viewport       viewport.Model
	messages       []string
	textarea       textarea.Model
	senderStyle    lipgloss.Style
	recipientStyle lipgloss.Style
	err            error
}

func New() model {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 280

	ta.SetWidth(30)
	ta.SetHeight(3)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	ta.ShowLineNumbers = false

	vp := viewport.New(30, 5)
	vp.SetContent(`RMX Chat
Type a message and press Enter to send.`)

	ta.KeyMap.InsertNewline.SetEnabled(false)

	return model{
		textarea:       ta,
		messages:       []string{},
		viewport:       vp,
		senderStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		recipientStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
		err:            nil,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	cmds = append(cmds, tiCmd, vpCmd)

	switch msg := msg.(type) {
	case ToggleFocusMsg:
		if m.textarea.Focused() {
			m.textarea.Blur()
		} else {
			m.textarea.Focus()
		}

	case RecvTextMsg:
		if msg.FromSelf {
			textMsg := fmt.Sprintf("You: %s", msg.Msg)
			m.messages = append(m.messages, m.senderStyle.Render(textMsg))
		} else {
			// Message from others
			textMsg := fmt.Sprintf("%s: %s", msg.DisplayName, msg.Msg)
			m.messages = append(m.messages, m.recipientStyle.Render(textMsg))
		}
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		m.viewport.GotoBottom()

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			fmt.Println(m.textarea.Value())
			return m, tea.Quit
		case tea.KeyEnter:
			m.viewport.SetContent(strings.Join(m.messages, "\n"))
			cmds = append(cmds, m.send)

			m.textarea.Reset()
			m.viewport.GotoBottom()
		}

	// We handle errors just like any other message
	case rmxerr.ErrMsg:
		m.err = msg
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	return fmt.Sprintf(
		"%s\n\n%s",
		m.viewport.View(),
		m.textarea.View(),
	) + "\n\n"
}

func (m model) send() tea.Msg {
	return SendMsg{Msg: m.textarea.Value()}
}
