package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"meshtastic-cli/internal/proto"
	"meshtastic-cli/internal/store"
)

type dmMode int

const (
	dmModeList dmMode = iota
	dmModeChat
)

type dmTab struct {
	mode          dmMode
	conversations []store.DMConversation
	messages      []store.Message
	cursor        int
	activeNode    uint32
	input         textinput.Model
}

func newDMTab() *dmTab {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.CharLimit = 228
	ti.Width = 60
	ti.Prompt = "» "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ColorGreen)
	ti.TextStyle = lipgloss.NewStyle().Foreground(ColorWhite)
	return &dmTab{input: ti}
}

func (t *dmTab) refresh(db *store.DB, myNode uint32) {
	if myNode == 0 {
		return
	}
	convos, err := db.GetDMConversations(myNode)
	if err == nil {
		t.conversations = convos
	}
	if t.cursor >= len(t.conversations) && len(t.conversations) > 0 {
		t.cursor = len(t.conversations) - 1
	}
}

func (t *dmTab) refreshMessages(db *store.DB, myNode uint32) {
	if myNode == 0 || t.activeNode == 0 {
		return
	}
	msgs, err := db.GetDMMessages(myNode, t.activeNode, 200)
	if err == nil {
		t.messages = msgs
	}
}

func (t *dmTab) view(m *Model, height int) string {
	if m.myNodeNum == 0 {
		return lipgloss.NewStyle().Foreground(ColorDim).Render("  Waiting for device info...")
	}

	if t.mode == dmModeChat {
		return t.viewChat(m, height)
	}
	return t.viewList(m, height)
}

func (t *dmTab) viewList(m *Model, height int) string {
	header := Title.Render("  Direct Messages")

	if len(t.conversations) == 0 {
		return header + "\n\n" +
			lipgloss.NewStyle().Foreground(ColorDim).Render("  No conversations yet. Press Enter on a node to start a DM.")
	}

	listHeight := height - 2 // header + blank line
	if listHeight < 1 {
		listHeight = 1
	}

	var lines []string
	lines = append(lines, header)
	lines = append(lines, "")

	for i, c := range t.conversations {
		if i >= listHeight {
			break
		}
		lines = append(lines, formatDMConversation(c, m, i == t.cursor))
	}

	return strings.Join(lines, "\n")
}

func formatDMConversation(c store.DMConversation, m *Model, selected bool) string {
	name := proto.NodeIDStr(c.NodeNum)
	if m.db != nil {
		if n := m.db.GetNodeName(c.NodeNum); n != "" {
			name = n
		}
	}

	ago := time.Since(time.UnixMilli(c.LastTimestamp))
	timeStr := ""
	if ago < time.Hour {
		timeStr = fmt.Sprintf("%dm", int(ago.Minutes()))
	} else if ago < 24*time.Hour {
		timeStr = fmt.Sprintf("%dh", int(ago.Hours()))
	} else {
		timeStr = time.UnixMilli(c.LastTimestamp).Format("Jan 02")
	}

	preview := c.LastMessage
	if len(preview) > 40 {
		preview = preview[:40] + "..."
	}

	unread := ""
	if c.UnreadCount > 0 {
		unread = lipgloss.NewStyle().Foreground(ColorGreen).Bold(true).Render(fmt.Sprintf(" (%d)", c.UnreadCount))
	}

	line := fmt.Sprintf("  %-16s  %-6s  %s%s", name, timeStr, preview, unread)

	if selected {
		return Selected.Width(m.width).Render(line)
	}
	return line
}

func (t *dmTab) viewChat(m *Model, height int) string {
	// Layout: header (1) + messages (height-3) + separator (1) + input (1)
	msgHeight := height - 3
	if msgHeight < 1 {
		msgHeight = 1
	}

	name := proto.NodeIDStr(t.activeNode)
	if m.db != nil {
		if n := m.db.GetNodeName(t.activeNode); n != "" {
			name = n
		}
	}
	header := Title.Render("  DM: "+name) + "  " +
		lipgloss.NewStyle().Foreground(ColorDim).Render("(Esc to go back)")

	var msgLines []string
	for _, msg := range t.messages {
		msgLines = append(msgLines, formatChatMsg(msg, m, m.width))
	}

	if len(msgLines) > msgHeight {
		msgLines = msgLines[len(msgLines)-msgHeight:]
	}
	for len(msgLines) < msgHeight {
		msgLines = append([]string{""}, msgLines...)
	}

	content := strings.Join(msgLines, "\n")
	separator := lipgloss.NewStyle().Foreground(ColorDim).Width(m.width).Render(strings.Repeat("─", m.width))
	inputLine := "  " + t.input.View()

	return header + "\n" + content + "\n" + separator + "\n" + inputLine
}
