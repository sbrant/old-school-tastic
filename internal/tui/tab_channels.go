package tui

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	pb "buf.build/gen/go/meshtastic/protobufs/protocolbuffers/go/meshtastic"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"meshtastic-cli/internal/proto"
)

type channelInfo struct {
	index    int32
	role     pb.Channel_Role
	name     string
	psk      []byte
	uplink   bool
	downlink bool
}

type channelEditField int

const (
	fieldName channelEditField = iota
	fieldPSK
	fieldRole
)

type channelsTab struct {
	channels []channelInfo
	cursor   int
	editing  bool
	editIdx  int
	field    channelEditField
	input    textinput.Model
	creating bool
	message  string // status message
	received int
}

func newChannelsTab() *channelsTab {
	ti := textinput.New()
	ti.CharLimit = 64
	ti.Width = 40
	ti.Prompt = "» "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ColorGreen)
	ti.TextStyle = lipgloss.NewStyle().Foreground(ColorWhite)
	return &channelsTab{input: ti}
}

func (t *channelsTab) addChannel(ch *pb.Channel) {
	if ch == nil {
		return
	}
	t.received++

	info := channelInfo{
		index: ch.GetIndex(),
		role:  ch.GetRole(),
	}
	if s := ch.GetSettings(); s != nil {
		info.name = s.GetName()
		info.psk = s.GetPsk()
		info.uplink = s.GetUplinkEnabled()
		info.downlink = s.GetDownlinkEnabled()
	}

	// Update existing or append
	for i, c := range t.channels {
		if c.index == info.index {
			t.channels[i] = info
			return
		}
	}
	t.channels = append(t.channels, info)
}

func (t *channelsTab) view(m *Model, height int) string {
	if len(t.channels) == 0 {
		return lipgloss.NewStyle().Foreground(ColorDim).Render("  No channels loaded. Press r to request from device.")
	}

	if t.editing {
		return t.viewEdit(m, height)
	}

	listHeight := height - 2
	if listHeight < 1 {
		listHeight = 1
	}

	hdr := Header.Render(fmt.Sprintf("  %-5s  %-10s  %-16s  %-40s  %s", "IDX", "ROLE", "NAME", "PSK", "UPLINK/DOWNLINK"))

	statusLine := ""
	if t.message != "" {
		statusLine = lipgloss.NewStyle().Foreground(ColorGreen).Render("  " + t.message)
	} else {
		statusLine = lipgloss.NewStyle().Foreground(ColorDim).Render("  Enter:edit  n:new channel  r:reload  d:disable")
	}

	var lines []string
	lines = append(lines, hdr)

	for i, ch := range t.channels {
		if i >= listHeight {
			break
		}
		lines = append(lines, formatChannelLine(ch, i == t.cursor, m.width))
	}

	lines = append(lines, statusLine)

	return strings.Join(lines, "\n")
}

func formatChannelLine(ch channelInfo, selected bool, width int) string {
	role := ch.role.String()
	roleColor := ColorDim
	switch ch.role {
	case pb.Channel_PRIMARY:
		roleColor = ColorGreen
	case pb.Channel_SECONDARY:
		roleColor = ColorCyan
	}

	pskStr := "(none)"
	if len(ch.psk) == 1 {
		if ch.psk[0] == 0 {
			pskStr = "(none)"
		} else if ch.psk[0] == 1 {
			pskStr = "default"
		} else {
			pskStr = fmt.Sprintf("simple%d", ch.psk[0])
		}
	} else if len(ch.psk) > 1 {
		pskStr = hex.EncodeToString(ch.psk)
		if len(pskStr) > 40 {
			pskStr = pskStr[:40]
		}
	}

	upDown := ""
	if ch.uplink || ch.downlink {
		parts := []string{}
		if ch.uplink {
			parts = append(parts, "up")
		}
		if ch.downlink {
			parts = append(parts, "down")
		}
		upDown = strings.Join(parts, "/")
	}

	roleStr := lipgloss.NewStyle().Foreground(roleColor).Render(fmt.Sprintf("%-10s", role))
	nameStr := lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(fmt.Sprintf("%-16s", ch.name))
	pskDisplay := lipgloss.NewStyle().Foreground(ColorOrange).Render(fmt.Sprintf("%-40s", pskStr))

	line := fmt.Sprintf("  %-5d  %s  %s  %s  %s", ch.index, roleStr, nameStr, pskDisplay, upDown)

	if selected {
		return Selected.Width(width).Render(line)
	}
	if ch.role == pb.Channel_DISABLED {
		return lipgloss.NewStyle().Foreground(ColorDim).Render(line)
	}
	return line
}

func (t *channelsTab) viewEdit(m *Model, height int) string {
	var lines []string

	title := "Edit Channel"
	if t.creating {
		title = "New Channel"
	}
	lines = append(lines, Title.Render("  "+title))
	lines = append(lines, "")

	ch := channelInfo{}
	if !t.creating && t.editIdx < len(t.channels) {
		ch = t.channels[t.editIdx]
	}

	fields := []struct {
		label  string
		value  string
		field  channelEditField
		active bool
	}{
		{"Name", ch.name, fieldName, t.field == fieldName},
		{"PSK", pskDisplayStr(ch.psk), fieldPSK, t.field == fieldPSK},
		{"Role", ch.role.String(), fieldRole, t.field == fieldRole},
	}

	for _, f := range fields {
		label := lipgloss.NewStyle().Foreground(ColorDim).Width(10).Render(f.label)
		if f.active {
			lines = append(lines, fmt.Sprintf("  %s  %s", label, t.input.View()))
		} else {
			val := lipgloss.NewStyle().Foreground(ColorWhite).Render(f.value)
			lines = append(lines, fmt.Sprintf("  %s  %s", label, val))
		}
	}

	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorDim).Render("  Tab:next field  Enter:save  g:generate random PSK  Esc:cancel"))

	return strings.Join(lines, "\n")
}

func pskDisplayStr(psk []byte) string {
	if len(psk) == 0 {
		return "(none)"
	}
	if len(psk) == 1 {
		if psk[0] == 1 {
			return "default"
		}
		return fmt.Sprintf("simple%d", psk[0])
	}
	return hex.EncodeToString(psk)
}

// generatePSK creates a random 32-byte (AES256) key
func generatePSK() []byte {
	key := make([]byte, 32)
	rand.Read(key)
	return key
}

// buildSetChannel creates the admin message to set a channel
func buildSetChannel(myNode uint32, idx int32, name string, psk []byte, role pb.Channel_Role) ([]byte, error) {
	ch := &pb.Channel{
		Index: idx,
		Role:  role,
		Settings: &pb.ChannelSettings{
			Name: name,
			Psk:  psk,
		},
	}

	return proto.EncodeAdminSetChannel(myNode, ch)
}
