package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var StatusReconnecting = lipgloss.NewStyle().
	Foreground(ColorYellow).
	Bold(true)

func renderStatusBar(m *Model) string {
	var status string
	if m.reconnecting {
		status = StatusReconnecting.Render("◌ RECONNECTING")
	} else if m.connected {
		status = StatusConnected.Render("● CONNECTED")
	} else {
		status = StatusDisconnected.Render("○ CONNECTING")
	}

	nodeCount := fmt.Sprintf("nodes:%d", m.nodeCount)
	pktCount := fmt.Sprintf("pkts:%d", m.packetCount)

	myNode := ""
	if m.myNodeNum != 0 {
		name := m.db.GetNodeName(m.myNodeNum)
		if name != "" {
			myNode = fmt.Sprintf("  me:%s", name)
		} else {
			myNode = fmt.Sprintf("  me:!%08x", m.myNodeNum)
		}
	}

	bbsStatus := ""
	if m.bbs != nil {
		bbsStatus = lipgloss.NewStyle().Foreground(ColorYellow).Bold(true).Render("  BBS")
	}

	left := fmt.Sprintf(" %s%s%s  %s  %s", status, myNode, bbsStatus, nodeCount, pktCount)
	right := " q:quit  ?:help "

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	bar := left + fmt.Sprintf("%*s", gap, "") + right
	return StatusBar.Width(m.width).Render(bar)
}
