package keymap

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type Mapping struct {
	CycleFocus key.Binding
	GoBack     key.Binding
	Quit       key.Binding
}

var DefaultMapping = Mapping{
	CycleFocus: key.NewBinding(
		key.WithKeys(tea.KeyTab.String()),
		key.WithHelp("tab", "cycle focus"),
	),
	GoBack: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "go back"),
	),
	Quit: key.NewBinding(
		key.WithKeys(tea.KeyCtrlC.String()),
		key.WithHelp("ctrl+c", "quit"),
	),
}
