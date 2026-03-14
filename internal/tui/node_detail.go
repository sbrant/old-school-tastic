package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"meshtastic-cli/internal/proto"
	"meshtastic-cli/internal/store"
)

func renderNodeDetail(n store.Node, width, height int) string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorGreen).
		Padding(1, 2).
		Width(min(60, width-4))

	title := Title.Render("NODE DETAIL")
	var lines []string
	lines = append(lines, title)
	lines = append(lines, "")

	addField := func(label string, value string) {
		if value == "" {
			return
		}
		l := lipgloss.NewStyle().Foreground(ColorDim).Width(18).Render(label)
		v := lipgloss.NewStyle().Foreground(ColorWhite).Render(value)
		lines = append(lines, "  "+l+v)
	}

	// Identity
	name := "?"
	if n.LongName.Valid && n.LongName.String != "" {
		name = n.LongName.String
	}
	addField("Name", name)

	if n.ShortName.Valid && n.ShortName.String != "" {
		addField("Short Name", n.ShortName.String)
	}

	addField("ID", proto.NodeIDStr(uint32(n.Num)))

	if n.UserID.Valid && n.UserID.String != "" {
		addField("User ID", n.UserID.String)
	}

	if n.HwModel.Valid {
		addField("Hardware", fmt.Sprintf("%d", n.HwModel.Int64))
	}

	if n.Role.Valid {
		addField("Role", fmt.Sprintf("%d", n.Role.Int64))
	}

	lines = append(lines, "")

	// Position
	if n.LatitudeI.Valid && n.LongitudeI.Valid {
		lat := float64(n.LatitudeI.Int64) / 1e7
		lon := float64(n.LongitudeI.Int64) / 1e7
		addField("Position", fmt.Sprintf("%.5f, %.5f", lat, lon))
	}
	if n.Altitude.Valid && n.Altitude.Int64 != 0 {
		addField("Altitude", fmt.Sprintf("%dm", n.Altitude.Int64))
	}

	// Metrics
	if n.SNR.Valid {
		addField("SNR", fmt.Sprintf("%.1f dB", n.SNR.Float64))
	}
	if n.BatteryLevel.Valid && n.BatteryLevel.Int64 > 0 {
		addField("Battery", fmt.Sprintf("%d%%", n.BatteryLevel.Int64))
	}
	if n.Voltage.Valid && n.Voltage.Float64 > 0 {
		addField("Voltage", fmt.Sprintf("%.2fV", n.Voltage.Float64))
	}
	if n.ChannelUtilization.Valid && n.ChannelUtilization.Float64 > 0 {
		addField("Channel Util", fmt.Sprintf("%.1f%%", n.ChannelUtilization.Float64))
	}
	if n.AirUtilTx.Valid && n.AirUtilTx.Float64 > 0 {
		addField("Air Util TX", fmt.Sprintf("%.1f%%", n.AirUtilTx.Float64))
	}

	// Connection
	if n.HopsAway.Valid {
		if n.HopsAway.Int64 == 0 {
			addField("Hops", "direct")
		} else {
			addField("Hops", fmt.Sprintf("%d", n.HopsAway.Int64))
		}
	}
	if n.Channel.Valid {
		addField("Channel", fmt.Sprintf("%d", n.Channel.Int64))
	}
	if n.LastHeard.Valid && n.LastHeard.Int64 > 0 {
		t := time.Unix(n.LastHeard.Int64, 0)
		addField("Last Heard", t.Format("2006-01-02 15:04:05"))
	}

	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorDim).Render("  Esc to close  •  Enter to DM"))

	box := boxStyle.Render(strings.Join(lines, "\n"))
	return lipgloss.Place(width, height-2, lipgloss.Center, lipgloss.Center, box)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
