package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"meshtastic-cli/internal/bbs"
)

type bbsTab struct {
	scroll int
}

func newBBSTab() *bbsTab {
	return &bbsTab{}
}

func (t *bbsTab) view(m *Model, height int) string {
	if m.bbs == nil {
		return lipgloss.NewStyle().Foreground(ColorDim).Render("  BBS mode not enabled. Start with --bbs flag.")
	}

	log := m.bbs.Log()

	headerLine := Title.Render("  BBS Activity Log") + "  " +
		lipgloss.NewStyle().Foreground(ColorDim).Render(fmt.Sprintf("%d entries", len(log)))

	listHeight := height - 1
	if listHeight < 1 {
		listHeight = 1
	}

	if len(log) == 0 {
		return headerLine + "\n" +
			lipgloss.NewStyle().Foreground(ColorDim).Render("  Waiting for incoming commands...")
	}

	var lines []string
	// Show most recent entries that fit
	start := 0
	if len(log) > listHeight {
		start = len(log) - listHeight
	}

	for _, entry := range log[start:] {
		lines = append(lines, formatBBSEntry(entry, m))
	}

	// Pad
	for len(lines) < listHeight {
		lines = append([]string{""}, lines...)
	}

	return headerLine + "\n" + strings.Join(lines, "\n")
}

func formatBBSEntry(entry bbs.LogEntry, m *Model) string {
	ts := entry.Timestamp.Format("15:04:05")
	name := entry.FromName
	if len(name) > 12 {
		name = name[:12]
	}

	tsStyle := lipgloss.NewStyle().Foreground(ColorDim)
	nameStyle := lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)
	cmdStyle := lipgloss.NewStyle().Foreground(ColorYellow)
	respStyle := lipgloss.NewStyle().Foreground(ColorGreen)

	// Truncate response for display
	resp := entry.Response
	if idx := strings.Index(resp, "\n"); idx > 0 {
		resp = resp[:idx] + "..."
	}
	if len(resp) > 60 {
		resp = resp[:60] + "..."
	}

	return fmt.Sprintf("  %s  %s  %s → %s",
		tsStyle.Render(ts),
		nameStyle.Render(fmt.Sprintf("%-12s", name)),
		cmdStyle.Render(fmt.Sprintf("%-20s", entry.Command)),
		respStyle.Render(resp),
	)
}
