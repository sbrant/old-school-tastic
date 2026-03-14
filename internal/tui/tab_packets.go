package tui

import (
	"fmt"
	"strings"
	"time"

	pb "buf.build/gen/go/meshtastic/protobufs/protocolbuffers/go/meshtastic"
	"github.com/charmbracelet/lipgloss"
	"meshtastic-cli/internal/proto"
)

type packetEntry struct {
	timestamp time.Time
	from      uint32
	to        uint32
	portnum   pb.PortNum
	payload   any
	rxSnr     float32
	rxRssi    int32
	hopLimit  uint32
	encrypted bool
	encLen    int
	raw       []byte

	// For non-mesh events (config, nodeinfo, channels, etc.)
	eventType string
	eventData string
}

type packetsTab struct {
	packets  []packetEntry
	cursor   int
	maxItems int
}

func newPacketsTab() *packetsTab {
	return &packetsTab{maxItems: 5000}
}

func (t *packetsTab) addPacket(pkt *proto.Packet) {
	fr := pkt.FromRadio
	now := time.Now()

	var entry packetEntry
	entry.timestamp = now
	entry.raw = pkt.Raw

	switch v := fr.PayloadVariant.(type) {
	case *pb.FromRadio_Packet:
		mp := v.Packet
		entry.from = mp.GetFrom()
		entry.to = mp.GetTo()
		entry.hopLimit = mp.GetHopLimit()
		entry.rxSnr = mp.GetRxSnr()
		entry.rxRssi = mp.GetRxRssi()
		if d := mp.GetDecoded(); d != nil {
			entry.portnum = d.GetPortnum()
			entry.payload = pkt.Payload
		} else if enc := mp.GetEncrypted(); enc != nil {
			entry.encrypted = true
			entry.encLen = len(enc)
		}

	case *pb.FromRadio_MyInfo:
		entry.eventType = "MY_INFO"
		entry.eventData = fmt.Sprintf("nodeNum:%d", v.MyInfo.GetMyNodeNum())

	case *pb.FromRadio_NodeInfo:
		entry.eventType = "NODE_INFO"
		name := ""
		if v.NodeInfo.GetUser() != nil {
			name = v.NodeInfo.GetUser().GetLongName()
		}
		entry.eventData = fmt.Sprintf("%s %q", proto.NodeIDStr(v.NodeInfo.GetNum()), name)

	case *pb.FromRadio_Config:
		entry.eventType = "CONFIG"
		entry.eventData = fmt.Sprintf("%T", v.Config.PayloadVariant)

	case *pb.FromRadio_ModuleConfig:
		entry.eventType = "MODULE_CONFIG"
		entry.eventData = fmt.Sprintf("%T", v.ModuleConfig.PayloadVariant)

	case *pb.FromRadio_Channel:
		entry.eventType = "CHANNEL"
		name := ""
		if v.Channel.GetSettings() != nil {
			name = v.Channel.GetSettings().GetName()
		}
		entry.eventData = fmt.Sprintf("idx:%d %q", v.Channel.GetIndex(), name)

	case *pb.FromRadio_ConfigCompleteId:
		entry.eventType = "CONFIG_COMPLETE"
		entry.eventData = fmt.Sprintf("id:%d", v.ConfigCompleteId)

	case *pb.FromRadio_LogRecord:
		entry.eventType = "LOG"
		lr := v.LogRecord
		entry.eventData = fmt.Sprintf("[%s] %s: %s", lr.GetLevel().String(), lr.GetSource(), lr.GetMessage())

	case *pb.FromRadio_Rebooted:
		entry.eventType = "REBOOTED"

	case *pb.FromRadio_Metadata:
		entry.eventType = "METADATA"
		entry.eventData = fmt.Sprintf("fw:%s hw:%s", v.Metadata.GetFirmwareVersion(), v.Metadata.GetHwModel().String())

	default:
		entry.eventType = "UNKNOWN"
		entry.eventData = fmt.Sprintf("%T", fr.PayloadVariant)
	}

	t.packets = append(t.packets, entry)
	if len(t.packets) > t.maxItems {
		t.packets = t.packets[len(t.packets)-t.maxItems:]
	}
	if t.cursor >= len(t.packets)-2 {
		t.cursor = len(t.packets) - 1
	}
}

func (t *packetsTab) view(m *Model, height int) string {
	if len(t.packets) == 0 {
		return lipgloss.NewStyle().Foreground(ColorDim).Render("  Waiting for packets...")
	}

	// 1 line for header, rest for packets
	listHeight := height - 1
	if listHeight < 1 {
		listHeight = 1
	}

	hdr := Header.Render(fmt.Sprintf("  %-8s  %-12s  %-12s  %-22s  %s", "TIME", "FROM", "TO", "TYPE", "DATA"))

	startIdx := 0
	if len(t.packets) > listHeight {
		halfView := listHeight / 2
		startIdx = t.cursor - halfView
		if startIdx < 0 {
			startIdx = 0
		}
		if startIdx > len(t.packets)-listHeight {
			startIdx = len(t.packets) - listHeight
		}
	}

	endIdx := startIdx + listHeight
	if endIdx > len(t.packets) {
		endIdx = len(t.packets)
	}

	var lines []string
	lines = append(lines, hdr)

	for i := startIdx; i < endIdx; i++ {
		p := t.packets[i]
		line := formatPacketLine(p, m, i == t.cursor)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func formatPacketLine(p packetEntry, m *Model, selected bool) string {
	ts := p.timestamp.Format("15:04:05")

	// Non-mesh event
	if p.eventType != "" {
		color := ColorConfig
		switch p.eventType {
		case "LOG":
			color = ColorDim
		case "NODE_INFO":
			color = ColorNodeInfo
		case "REBOOTED":
			color = ColorRed
		case "CONFIG_COMPLETE":
			color = ColorGreen
		}

		typeStr := lipgloss.NewStyle().Foreground(color).Render(fmt.Sprintf("%-22s", p.eventType))
		line := fmt.Sprintf("  %-8s  %-12s  %-12s  %s  %s", ts, "", "", typeStr, p.eventData)

		if selected {
			return Selected.Width(m.width).Render(line)
		}
		return line
	}

	// Mesh packet
	from := nodeLabel(p.from, m)
	to := nodeLabel(p.to, m)

	var portName, data string
	var color lipgloss.Color

	if p.encrypted {
		portName = "ENCRYPTED"
		data = fmt.Sprintf("(%d bytes)", p.encLen)
		color = ColorEncrypted
	} else {
		portName = proto.PortnumName(p.portnum)
		data = payloadSummary(p.payload, m.opts.Fahrenheit)
		color = PortnumColor(portName)
	}

	if len(from) > 12 {
		from = from[:12]
	}
	if len(to) > 12 {
		to = to[:12]
	}

	portStyle := lipgloss.NewStyle().Foreground(color)
	portStr := portStyle.Render(fmt.Sprintf("%-22s", portName))

	meta := ""
	if p.rxSnr != 0 || p.rxRssi != 0 {
		meta = lipgloss.NewStyle().Foreground(ColorDim).Render(fmt.Sprintf(" SNR:%.0f RSSI:%d", p.rxSnr, p.rxRssi))
	}

	line := fmt.Sprintf("  %-8s  %-12s  %-12s  %s  %s%s", ts, from, to, portStr, data, meta)

	if selected {
		return Selected.Width(m.width).Render(line)
	}
	return line
}

func payloadSummary(payload any, fahrenheit bool) string {
	switch v := payload.(type) {
	case string:
		if len(v) > 50 {
			return fmt.Sprintf("%q", v[:50]+"...")
		}
		return fmt.Sprintf("%q", v)
	case *pb.Position:
		lat := float64(v.GetLatitudeI()) / 1e7
		lon := float64(v.GetLongitudeI()) / 1e7
		return fmt.Sprintf("%.4f, %.4f", lat, lon)
	case *pb.User:
		return fmt.Sprintf("%s (%s)", v.GetLongName(), v.GetShortName())
	case *pb.Telemetry:
		if dm := v.GetDeviceMetrics(); dm != nil {
			parts := []string{}
			if dm.GetBatteryLevel() > 0 {
				parts = append(parts, fmt.Sprintf("bat:%d%%", dm.GetBatteryLevel()))
			}
			if dm.GetVoltage() > 0 {
				parts = append(parts, fmt.Sprintf("%.1fV", dm.GetVoltage()))
			}
			if dm.GetChannelUtilization() > 0 {
				parts = append(parts, fmt.Sprintf("ch:%.1f%%", dm.GetChannelUtilization()))
			}
			return strings.Join(parts, " ")
		}
		if em := v.GetEnvironmentMetrics(); em != nil {
			parts := []string{}
			if em.GetTemperature() > 0 {
				if fahrenheit {
					parts = append(parts, fmt.Sprintf("%.1f°F", em.GetTemperature()*9/5+32))
				} else {
					parts = append(parts, fmt.Sprintf("%.1f°C", em.GetTemperature()))
				}
			}
			if em.GetRelativeHumidity() > 0 {
				parts = append(parts, fmt.Sprintf("hum:%.0f%%", em.GetRelativeHumidity()))
			}
			return strings.Join(parts, " ")
		}
		return "telemetry"
	case *pb.Routing:
		if re := v.GetErrorReason(); re != pb.Routing_NONE {
			return re.String()
		}
		return "ACK"
	case *pb.RouteDiscovery:
		return fmt.Sprintf("%d hops", len(v.GetRoute()))
	case *pb.NeighborInfo:
		return fmt.Sprintf("%d neighbors", len(v.GetNeighbors()))
	case []byte:
		return fmt.Sprintf("(%d bytes)", len(v))
	default:
		return ""
	}
}

func nodeLabel(num uint32, m *Model) string {
	if num == 0xFFFFFFFF {
		return "broadcast"
	}
	if m.db != nil {
		if name := m.db.GetNodeName(num); name != "" {
			return name
		}
	}
	return proto.NodeIDStr(num)
}
