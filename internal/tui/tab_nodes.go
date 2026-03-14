package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"meshtastic-cli/internal/proto"
	"meshtastic-cli/internal/store"
)

type nodesTab struct {
	nodes  []store.Node
	cursor int
}

func newNodesTab() *nodesTab {
	return &nodesTab{}
}

func (t *nodesTab) refresh(db *store.DB) {
	nodes, err := db.GetAllNodes()
	if err == nil {
		t.nodes = nodes
	}
	if t.cursor >= len(t.nodes) && len(t.nodes) > 0 {
		t.cursor = len(t.nodes) - 1
	}
}

func (t *nodesTab) view(m *Model, height int) string {
	if len(t.nodes) == 0 {
		return lipgloss.NewStyle().Foreground(ColorDim).Render("  No nodes discovered yet...")
	}

	listHeight := height - 1
	if listHeight < 1 {
		listHeight = 1
	}

	hdr := Header.Render(fmt.Sprintf("  %-16s  %-4s  %-12s  %-6s  %-5s  %-6s  %-7s  %s",
		"NAME", "ABBR", "ID", "SNR", "BAT", "CH-UTL", "HOPS", "LAST HEARD"))

	startIdx := 0
	if len(t.nodes) > listHeight {
		halfView := listHeight / 2
		startIdx = t.cursor - halfView
		if startIdx < 0 {
			startIdx = 0
		}
		if startIdx > len(t.nodes)-listHeight {
			startIdx = len(t.nodes) - listHeight
		}
	}

	endIdx := startIdx + listHeight
	if endIdx > len(t.nodes) {
		endIdx = len(t.nodes)
	}

	var lines []string
	lines = append(lines, hdr)

	for i := startIdx; i < endIdx; i++ {
		n := t.nodes[i]
		line := formatNodeLine(n, i == t.cursor, m.width)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

var (
	nameStyle     = lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	shortStyle    = lipgloss.NewStyle().Foreground(ColorCyan)
	idStyle       = lipgloss.NewStyle().Foreground(ColorDim)
	snrStyle      = lipgloss.NewStyle().Foreground(ColorYellow)
	batStyle      = lipgloss.NewStyle().Foreground(ColorOrange)
	chUtilStyle   = lipgloss.NewStyle().Foreground(ColorCyan)
	hopsStyle     = lipgloss.NewStyle().Foreground(ColorPurple)
	lastHeardStyle = lipgloss.NewStyle().Foreground(ColorWhite)
	staleStyle    = lipgloss.NewStyle().Foreground(ColorDim)
)

func formatNodeLine(n store.Node, selected bool, width int) string {
	stale := false
	if n.LastHeard.Valid {
		ago := time.Since(time.Unix(n.LastHeard.Int64, 0))
		if ago > 2*time.Hour {
			stale = true
		}
	}

	name := "?"
	if n.LongName.Valid && n.LongName.String != "" {
		name = n.LongName.String
	}
	if len(name) > 16 {
		name = name[:16]
	}

	short := ""
	if n.ShortName.Valid {
		short = n.ShortName.String
	}
	if len(short) > 4 {
		short = short[:4]
	}

	id := proto.NodeIDStr(uint32(n.Num))

	snr := ""
	if n.SNR.Valid {
		snr = fmt.Sprintf("%.1f", n.SNR.Float64)
	}

	bat := ""
	if n.BatteryLevel.Valid && n.BatteryLevel.Int64 > 0 {
		bat = fmt.Sprintf("%d%%", n.BatteryLevel.Int64)
	}

	chUtil := ""
	if n.ChannelUtilization.Valid && n.ChannelUtilization.Float64 > 0 {
		chUtil = fmt.Sprintf("%.1f%%", n.ChannelUtilization.Float64)
	}

	hops := ""
	if n.HopsAway.Valid {
		if n.HopsAway.Int64 == 0 {
			hops = "direct"
		} else {
			hops = fmt.Sprintf("%d", n.HopsAway.Int64)
		}
	}

	lastHeard := ""
	if n.LastHeard.Valid && n.LastHeard.Int64 > 0 {
		t := time.Unix(n.LastHeard.Int64, 0)
		ago := time.Since(t)
		if ago < time.Minute {
			lastHeard = fmt.Sprintf("%ds ago", int(ago.Seconds()))
		} else if ago < time.Hour {
			lastHeard = fmt.Sprintf("%dm ago", int(ago.Minutes()))
		} else if ago < 24*time.Hour {
			lastHeard = fmt.Sprintf("%dh ago", int(ago.Hours()))
		} else {
			lastHeard = t.Format("Jan 02")
		}
	}

	if selected {
		line := fmt.Sprintf("  %-16s  %-4s  %-12s  %-6s  %-5s  %-6s  %-7s  %s",
			name, short, id, snr, bat, chUtil, hops, lastHeard)
		return Selected.Width(width).Render(line)
	}

	if stale {
		line := fmt.Sprintf("  %-16s  %-4s  %-12s  %-6s  %-5s  %-6s  %-7s  %s",
			name, short, id, snr, bat, chUtil, hops, lastHeard)
		return staleStyle.Render(line)
	}

	// Colorized fields
	line := fmt.Sprintf("  %s  %s  %s  %s  %s  %s  %s  %s",
		nameStyle.Width(16).Render(name),
		shortStyle.Width(4).Render(short),
		idStyle.Width(12).Render(id),
		snrStyle.Width(6).Render(snr),
		batStyle.Width(5).Render(bat),
		chUtilStyle.Width(6).Render(chUtil),
		hopsStyle.Width(7).Render(hops),
		lastHeardStyle.Render(lastHeard),
	)

	return line
}
