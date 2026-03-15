package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"meshtastic-cli/internal/store"
)

type chatEntry struct {
	index   int    // channel index
	name    string // channel name
	isDM    bool
	dmNode  uint32
	dmName  string
	unread  int
	lastMsg string
	lastTs  int64
}

type chatTab struct {
	entries    []chatEntry
	cursor     int
	messages   []store.Message
	input      textinput.Model
	focusRight bool // true when chat panel has focus (typing mode)
}

func newChatTab() *chatTab {
	ti := textinput.New()
	ti.Placeholder = "Message... (/dm <name> <msg> for DM)"
	ti.CharLimit = 228
	ti.Width = 60
	ti.Prompt = "» "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ColorGreen)
	ti.TextStyle = lipgloss.NewStyle().Foreground(ColorWhite)
	return &chatTab{input: ti}
}

func (t *chatTab) refresh(m *Model) {
	t.entries = nil

	// Add channels
	for i, name := range m.channels {
		if i == 0 {
			if name == "" {
				name = "Primary"
			}
		} else if name == "" {
			// Check if channel is active
			active := false
			for _, ch := range m.channelsUI.channels {
				if int(ch.index) == i && ch.role != 0 { // not DISABLED
					active = true
					if ch.name != "" {
						name = ch.name
					}
					break
				}
			}
			if !active {
				continue
			}
			if name == "" {
				name = fmt.Sprintf("Channel %d", i)
			}
		}
		t.entries = append(t.entries, chatEntry{
			index: i,
			name:  name,
		})
	}

	// Add DM conversations
	if m.myNodeNum != 0 {
		convos, err := m.db.GetDMConversations(m.myNodeNum)
		if err == nil {
			for _, c := range convos {
				nodeName := m.db.GetNodeName(c.NodeNum)
				if nodeName == "" {
					nodeName = fmt.Sprintf("!%08x", c.NodeNum)
				}
				t.entries = append(t.entries, chatEntry{
					isDM:    true,
					dmNode:  c.NodeNum,
					dmName:  nodeName,
					name:    "DM: " + nodeName,
					unread:  c.UnreadCount,
					lastMsg: c.LastMessage,
					lastTs:  c.LastTimestamp,
				})
			}
		}
	}

	if t.cursor >= len(t.entries) && len(t.entries) > 0 {
		t.cursor = len(t.entries) - 1
	}

	// Load messages for selected entry
	t.loadMessages(m)
}

func (t *chatTab) loadMessages(m *Model) {
	if t.cursor >= len(t.entries) {
		t.messages = nil
		return
	}
	entry := t.entries[t.cursor]
	if entry.isDM {
		msgs, err := m.db.GetDMMessages(m.myNodeNum, entry.dmNode, 200)
		if err == nil {
			t.messages = msgs
		}
	} else {
		msgs, err := m.db.GetMessages(entry.index, 200)
		if err == nil {
			t.messages = msgs
		}
	}
}

func (t *chatTab) selectedChannel() (isDM bool, chIndex int, dmNode uint32) {
	if t.cursor >= len(t.entries) {
		return false, 0, 0
	}
	e := t.entries[t.cursor]
	return e.isDM, e.index, e.dmNode
}

func (t *chatTab) view(m *Model, height int) string {
	// Layout: left panel (20 chars) | right panel (rest)
	leftWidth := 22
	rightWidth := m.width - leftWidth - 1 // 1 for separator
	if rightWidth < 20 {
		rightWidth = 20
	}

	// Build left panel
	leftLines := t.buildLeftPanel(m, height, leftWidth)

	// Build right panel
	rightLines := t.buildRightPanel(m, height, rightWidth)

	// Combine with separator
	sep := lipgloss.NewStyle().Foreground(ColorDim).Render("│")
	var combined []string
	for i := 0; i < height; i++ {
		left := ""
		right := ""
		if i < len(leftLines) {
			left = leftLines[i]
		}
		if i < len(rightLines) {
			right = rightLines[i]
		}
		combined = append(combined, left+sep+right)
	}

	return strings.Join(combined, "\n")
}

func (t *chatTab) buildLeftPanel(m *Model, height, width int) []string {
	var lines []string

	hdr := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true).Width(width).Render(" Channels")
	lines = append(lines, hdr)

	for i, e := range t.entries {
		if i >= height-1 {
			break
		}
		line := t.formatLeftEntry(e, i == t.cursor && !t.focusRight, width)
		lines = append(lines, line)
	}

	// Pad
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}

	return lines
}

func (t *chatTab) formatLeftEntry(e chatEntry, selected bool, width int) string {
	name := e.name
	if len(name) > width-4 {
		name = name[:width-4]
	}

	unread := ""
	if e.unread > 0 {
		unread = lipgloss.NewStyle().Foreground(ColorGreen).Bold(true).Render(fmt.Sprintf(" %d", e.unread))
	}

	line := fmt.Sprintf(" %s%s", name, unread)

	if selected {
		return Selected.Width(width).Render(line)
	}

	if e.isDM {
		return lipgloss.NewStyle().Foreground(ColorCyan).Width(width).Render(line)
	}
	return lipgloss.NewStyle().Foreground(ColorWhite).Width(width).Render(line)
}

func (t *chatTab) buildRightPanel(m *Model, height, width int) []string {
	// Layout: header (1) + messages (height-3) + separator (1) + input (1)
	msgHeight := height - 3
	if msgHeight < 1 {
		msgHeight = 1
	}

	// Header
	chName := ""
	if t.cursor < len(t.entries) {
		chName = t.entries[t.cursor].name
	}
	hdr := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true).Render(
		fmt.Sprintf(" %s (%d msgs)", chName, len(t.messages)))
	hdr = lipgloss.NewStyle().Width(width).Render(hdr)

	// Messages
	var msgLines []string
	for _, msg := range t.messages {
		line := formatChatMsg(msg, m, width)
		msgLines = append(msgLines, line)
	}
	if len(msgLines) > msgHeight {
		msgLines = msgLines[len(msgLines)-msgHeight:]
	}
	for len(msgLines) < msgHeight {
		msgLines = append([]string{strings.Repeat(" ", width)}, msgLines...)
	}

	// Separator
	sepLine := lipgloss.NewStyle().Foreground(ColorDim).Width(width).Render(strings.Repeat("─", width))

	// Input
	inputLine := " " + t.input.View()
	inputLine = lipgloss.NewStyle().Width(width).Render(inputLine)

	var lines []string
	lines = append(lines, hdr)
	lines = append(lines, msgLines...)
	lines = append(lines, sepLine)
	lines = append(lines, inputLine)

	return lines
}

func formatChatMsg(msg store.Message, m *Model, width int) string {
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

	nameColor := ColorCyan
	if uint32(msg.FromNode) == m.myNodeNum {
		nameColor = ColorGreen
	}

	status := ""
	if msg.Status == "pending" {
		status = lipgloss.NewStyle().Foreground(ColorDim).Render(" ...")
	} else if msg.Status == "error" {
		status = lipgloss.NewStyle().Foreground(ColorRed).Render(" err")
	}

	tsStr := lipgloss.NewStyle().Foreground(ColorDim).Render(ts)
	nameStr := lipgloss.NewStyle().Foreground(nameColor).Bold(true).Render(name)
	textStr := lipgloss.NewStyle().Foreground(ColorWhite).Render(msg.Text)

	return fmt.Sprintf(" %s %s: %s%s", tsStr, nameStr, textStr, status)
}
