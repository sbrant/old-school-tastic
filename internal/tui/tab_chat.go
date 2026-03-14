package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"meshtastic-cli/internal/store"
)

type chatTab struct {
	messages []store.Message
	input    textinput.Model
	channel  int
	scroll   int
}

func newChatTab() *chatTab {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.CharLimit = 228
	ti.Width = 60
	ti.Prompt = "» "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ColorGreen)
	ti.TextStyle = lipgloss.NewStyle().Foreground(ColorWhite)
	return &chatTab{input: ti}
}

func (t *chatTab) refresh(db *store.DB) {
	msgs, err := db.GetMessages(t.channel, 200)
	if err == nil {
		t.messages = msgs
	}
}

func (t *chatTab) view(m *Model, height int) string {
	// Layout: header (1) + messages (height-3) + separator (1) + input (1)
	msgHeight := height - 3
	if msgHeight < 1 {
		msgHeight = 1
	}

	chLabel := fmt.Sprintf("Channel %d", t.channel)
	if len(m.channels) > t.channel {
		name := m.channels[t.channel]
		if name != "" {
			chLabel = fmt.Sprintf("#%s (ch:%d)", name, t.channel)
		}
	}
	header := Title.Render("  "+chLabel) + "  " +
		lipgloss.NewStyle().Foreground(ColorDim).Render(fmt.Sprintf("%d messages", len(t.messages)))

	var msgLines []string
	for _, msg := range t.messages {
		msgLines = append(msgLines, formatChatMessage(msg, m))
	}

	if len(msgLines) > msgHeight {
		msgLines = msgLines[len(msgLines)-msgHeight:]
	}
	for len(msgLines) < msgHeight {
		msgLines = append([]string{""}, msgLines...)
	}

	content := strings.Join(msgLines, "\n")
	separator := lipgloss.NewStyle().Foreground(ColorDim).Render(strings.Repeat("─", m.width))
	inputLine := "  " + t.input.View()

	return header + "\n" + content + "\n" + separator + "\n" + inputLine
}

func formatChatMessage(msg store.Message, m *Model) string {
	ts := time.UnixMilli(msg.Timestamp).Format("15:04")
	name := "?"
	if m.db != nil {
		if n := m.db.GetNodeName(uint32(msg.FromNode)); n != "" {
			name = n
		}
	}
	if name == "?" {
		name = fmt.Sprintf("!%08x", msg.FromNode)
	}

	// Color the sender name
	nameColor := ColorCyan
	if uint32(msg.FromNode) == m.myNodeNum {
		nameColor = ColorGreen
	}

	status := ""
	if msg.Status == "pending" {
		status = lipgloss.NewStyle().Foreground(ColorDim).Render(" ⏳")
	} else if msg.Status == "error" {
		status = lipgloss.NewStyle().Foreground(ColorRed).Render(" ✗")
	}

	tsStr := lipgloss.NewStyle().Foreground(ColorDim).Render(ts)
	nameStr := lipgloss.NewStyle().Foreground(nameColor).Bold(true).Render(name)
	textStr := lipgloss.NewStyle().Foreground(ColorWhite).Render(msg.Text)

	return fmt.Sprintf("  %s  %s: %s%s", tsStr, nameStr, textStr, status)
}
