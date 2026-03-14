package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var helpContent = []struct {
	key  string
	desc string
}{
	{"1-5", "Switch tabs"},
	{"j/↓", "Move down"},
	{"k/↑", "Move up"},
	{"g", "Go to top"},
	{"G", "Go to bottom"},
	{"^D/^U", "Page down/up"},
	{"Enter", "Select / start typing"},
	{"Esc", "Back / stop typing"},
	{"r", "Reload config (config tab)"},
	{"?", "Toggle help"},
	{"q", "Quit"},
}

func renderHelp(width, height int) string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorGreen).
		Padding(1, 2).
		Width(40)

	title := Title.Render("KEYBINDINGS")
	var lines []string
	lines = append(lines, title)
	lines = append(lines, "")

	for _, h := range helpContent {
		key := lipgloss.NewStyle().Foreground(ColorCyan).Bold(true).Width(8).Render(h.key)
		desc := lipgloss.NewStyle().Foreground(ColorWhite).Render(h.desc)
		lines = append(lines, "  "+key+"  "+desc)
	}

	box := boxStyle.Render(strings.Join(lines, "\n"))
	return lipgloss.Place(width, height-2, lipgloss.Center, lipgloss.Center, box)
}
