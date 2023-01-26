package styles

import "github.com/charmbracelet/lipgloss"

const (
	// In real life situations we'd adjust the document to fit the width we've
	// detected. In the case of this example we're hardcoding the width, and
	// later using the detected width only to truncate in order to avoid jaggy
	// wrapping.
	Width = 72
)

// https://github.com/inngest/inngest/blob/main/pkg/cli/styles.go
var (
	Color   = lipgloss.AdaptiveColor{Light: "#111222", Dark: "#FAFAFA"}
	Primary = lipgloss.Color("#4636f5")
	Green   = lipgloss.Color("#9dcc3a")
	Red     = lipgloss.Color("#ff0000")
	White   = lipgloss.Color("#ffffff")
	Black   = lipgloss.Color("#000000")
	Orange  = lipgloss.Color("#D3A347")

	TextStyle = lipgloss.NewStyle().Foreground(Color)
	BoldStyle = TextStyle.Copy().Bold(true)

	BaseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))

	// Status Bar.
	StatusNugget = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Padding(0, 1)
	PingStyle = StatusNugget.Copy().
			Background(lipgloss.Color("#e783f2")).
			Align(lipgloss.Right)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#343433", Dark: "#C1C6B2"}).
			Background(lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#353533"})

	StatusStyle = lipgloss.NewStyle().
			Inherit(StatusBarStyle).
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#FF5F87")).
			Padding(0, 1).
			MarginRight(1)

	StatusText = lipgloss.NewStyle().Inherit(StatusBarStyle)

	MessageText = lipgloss.NewStyle().Align(lipgloss.Left)

	HelpMenu = lipgloss.NewStyle().Align(lipgloss.Center).PaddingTop(2)
	// Page
	DocStyle = lipgloss.NewStyle().Padding(1, 2, 1, 2)
)

// RenderError returns a formatted error string.
func RenderError(msg string) string {
	// Error applies styles to an error message
	err := lipgloss.NewStyle().Background(Red).Foreground(White).Bold(true).Padding(0, 1).Render("Error")
	content := lipgloss.NewStyle().Bold(true).Padding(0, 1).Render(msg)
	return err + content
}
