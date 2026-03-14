package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Cyberpunk palette
	ColorGreen   = lipgloss.Color("#00ff9f")
	ColorCyan    = lipgloss.Color("#00bfff")
	ColorOrange  = lipgloss.Color("#ff9f00")
	ColorPurple  = lipgloss.Color("#bf00ff")
	ColorGray    = lipgloss.Color("#666666")
	ColorRed     = lipgloss.Color("#ff0040")
	ColorBlue    = lipgloss.Color("#8080ff")
	ColorYellow  = lipgloss.Color("#ffff00")
	ColorDim     = lipgloss.Color("#3d444c")
	ColorBg      = lipgloss.Color("#0a0e14")
	ColorPanelBg = lipgloss.Color("#0d1117")
	ColorWhite   = lipgloss.Color("#e6edf3")

	// Packet type colors
	ColorMessage   = ColorGreen
	ColorPosition  = ColorCyan
	ColorTelemetry = ColorOrange
	ColorNodeInfo  = ColorPurple
	ColorRouting   = ColorGray
	ColorEncrypted = ColorRed
	ColorConfig    = ColorBlue

	// Styles
	TabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorGreen).
			Background(lipgloss.Color("#1a2a1a")).
			Padding(0, 1)

	TabInactive = lipgloss.NewStyle().
			Foreground(ColorDim).
			Padding(0, 1)

	StatusBar = lipgloss.NewStyle().
			Foreground(ColorDim).
			Background(lipgloss.Color("#161b22"))

	StatusConnected = lipgloss.NewStyle().
			Foreground(ColorGreen).
			Bold(true)

	StatusDisconnected = lipgloss.NewStyle().
				Foreground(ColorRed).
				Bold(true)

	Selected = lipgloss.NewStyle().
			Background(lipgloss.Color("#1a2a1a")).
			Foreground(ColorGreen)

	Header = lipgloss.NewStyle().
		Foreground(ColorDim).
		Bold(true)

	Title = lipgloss.NewStyle().
		Foreground(ColorGreen).
		Bold(true)
)

func PortnumColor(portnum string) lipgloss.Color {
	switch portnum {
	case "TEXT_MESSAGE_APP":
		return ColorMessage
	case "POSITION_APP":
		return ColorPosition
	case "TELEMETRY_APP":
		return ColorTelemetry
	case "NODEINFO_APP":
		return ColorNodeInfo
	case "ROUTING_APP":
		return ColorRouting
	case "ADMIN_APP":
		return ColorConfig
	default:
		return ColorDim
	}
}
