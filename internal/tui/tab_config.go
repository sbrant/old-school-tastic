package tui

import (
	"fmt"
	"strings"

	pb "buf.build/gen/go/meshtastic/protobufs/protocolbuffers/go/meshtastic"
	"github.com/charmbracelet/lipgloss"
)

type configEntry struct {
	section string
	key     string
	value   string
}

type configTab struct {
	entries  []configEntry
	cursor   int
	received int      // count of config packets received
	types    []string // type names of received packets for debugging
}

func newConfigTab() *configTab {
	return &configTab{}
}

func (t *configTab) add(section, key, value string) {
	// Update existing entry if same section+key
	for i, e := range t.entries {
		if e.section == section && e.key == key {
			t.entries[i].value = value
			return
		}
	}
	t.entries = append(t.entries, configEntry{section, key, value})
}

func (t *configTab) addConfig(cfg *pb.Config) {
	if cfg == nil {
		return
	}

	t.received++
	t.types = append(t.types, fmt.Sprintf("C:%T", cfg.GetPayloadVariant()))

	switch v := cfg.PayloadVariant.(type) {
	case *pb.Config_Device:
		if d := v.Device; d != nil {
			t.add("Device", "Role", d.GetRole().String())
			t.add("Device", "SerialEnabled", fmt.Sprintf("%v", d.GetSerialEnabled()))
			t.add("Device", "RebroadcastMode", d.GetRebroadcastMode().String())
			if d.GetNodeInfoBroadcastSecs() > 0 {
				t.add("Device", "NodeInfoBroadcastSecs", fmt.Sprintf("%d", d.GetNodeInfoBroadcastSecs()))
			}
		}
	case *pb.Config_Position:
		if p := v.Position; p != nil {
			t.add("Position", "PositionBroadcastSecs", fmt.Sprintf("%d", p.GetPositionBroadcastSecs()))
			t.add("Position", "GpsEnabled", fmt.Sprintf("%v", p.GetGpsEnabled()))
			t.add("Position", "FixedPosition", fmt.Sprintf("%v", p.GetFixedPosition()))
			if p.GetPositionFlags() > 0 {
				t.add("Position", "PositionFlags", fmt.Sprintf("%d", p.GetPositionFlags()))
			}
		}
	case *pb.Config_Power:
		if p := v.Power; p != nil {
			t.add("Power", "IsPowerSaving", fmt.Sprintf("%v", p.GetIsPowerSaving()))
			if p.GetOnBatteryShutdownAfterSecs() > 0 {
				t.add("Power", "ShutdownAfterSecs", fmt.Sprintf("%d", p.GetOnBatteryShutdownAfterSecs()))
			}
			if p.GetWaitBluetoothSecs() > 0 {
				t.add("Power", "WaitBluetoothSecs", fmt.Sprintf("%d", p.GetWaitBluetoothSecs()))
			}
			if p.GetLsSecs() > 0 {
				t.add("Power", "LightSleepSecs", fmt.Sprintf("%d", p.GetLsSecs()))
			}
			if p.GetMinWakeSecs() > 0 {
				t.add("Power", "MinWakeSecs", fmt.Sprintf("%d", p.GetMinWakeSecs()))
			}
		}
	case *pb.Config_Network:
		if n := v.Network; n != nil {
			t.add("Network", "WifiEnabled", fmt.Sprintf("%v", n.GetWifiEnabled()))
			if n.GetWifiSsid() != "" {
				t.add("Network", "WifiSSID", n.GetWifiSsid())
			}
			t.add("Network", "NtpServer", n.GetNtpServer())
			t.add("Network", "EthEnabled", fmt.Sprintf("%v", n.GetEthEnabled()))
		}
	case *pb.Config_Display:
		if d := v.Display; d != nil {
			t.add("Display", "ScreenOnSecs", fmt.Sprintf("%d", d.GetScreenOnSecs()))
			t.add("Display", "GpsFormat", d.GetGpsFormat().String())
			t.add("Display", "Units", d.GetUnits().String())
			t.add("Display", "OledType", d.GetOled().String())
			t.add("Display", "FlipScreen", fmt.Sprintf("%v", d.GetFlipScreen()))
		}
	case *pb.Config_Lora:
		if l := v.Lora; l != nil {
			t.add("LoRa", "Region", l.GetRegion().String())
			t.add("LoRa", "ModemPreset", l.GetModemPreset().String())
			t.add("LoRa", "UsePreset", fmt.Sprintf("%v", l.GetUsePreset()))
			t.add("LoRa", "HopLimit", fmt.Sprintf("%d", l.GetHopLimit()))
			t.add("LoRa", "TxEnabled", fmt.Sprintf("%v", l.GetTxEnabled()))
			t.add("LoRa", "TxPower", fmt.Sprintf("%d", l.GetTxPower()))
			if l.GetBandwidth() > 0 {
				t.add("LoRa", "Bandwidth", fmt.Sprintf("%d", l.GetBandwidth()))
			}
			if l.GetSpreadFactor() > 0 {
				t.add("LoRa", "SpreadFactor", fmt.Sprintf("%d", l.GetSpreadFactor()))
			}
			if l.GetCodingRate() > 0 {
				t.add("LoRa", "CodingRate", fmt.Sprintf("%d", l.GetCodingRate()))
			}
			if l.GetChannelNum() > 0 {
				t.add("LoRa", "ChannelNum", fmt.Sprintf("%d", l.GetChannelNum()))
			}
		}
	case *pb.Config_Bluetooth:
		if b := v.Bluetooth; b != nil {
			t.add("Bluetooth", "Enabled", fmt.Sprintf("%v", b.GetEnabled()))
			t.add("Bluetooth", "Mode", b.GetMode().String())
			if b.GetFixedPin() > 0 {
				t.add("Bluetooth", "FixedPin", fmt.Sprintf("%d", b.GetFixedPin()))
			}
		}
	case *pb.Config_Security:
		if s := v.Security; s != nil {
			t.add("Security", "IsManaged", fmt.Sprintf("%v", s.GetIsManaged()))
			t.add("Security", "SerialEnabled", fmt.Sprintf("%v", s.GetSerialEnabled()))
			t.add("Security", "DebugLogApiEnabled", fmt.Sprintf("%v", s.GetDebugLogApiEnabled()))
			t.add("Security", "AdminChannelEnabled", fmt.Sprintf("%v", s.GetAdminChannelEnabled()))
		}
	case *pb.Config_DeviceUi:
		if d := v.DeviceUi; d != nil {
			t.add("DeviceUI", "ScreenBrightness", fmt.Sprintf("%d", d.GetScreenBrightness()))
			t.add("DeviceUI", "ScreenTimeout", fmt.Sprintf("%d", d.GetScreenTimeout()))
			t.add("DeviceUI", "ScreenLock", fmt.Sprintf("%v", d.GetScreenLock()))
			t.add("DeviceUI", "Theme", d.GetTheme().String())
			t.add("DeviceUI", "Language", d.GetLanguage().String())
			t.add("DeviceUI", "AlertEnabled", fmt.Sprintf("%v", d.GetAlertEnabled()))
		}
	}
}

func (t *configTab) addModuleConfig(cfg *pb.ModuleConfig) {
	if cfg == nil {
		return
	}

	t.received++
	t.types = append(t.types, fmt.Sprintf("M:%T", cfg.GetPayloadVariant()))

	switch v := cfg.PayloadVariant.(type) {
	case *pb.ModuleConfig_Mqtt:
		if m := v.Mqtt; m != nil {
			t.add("MQTT", "Enabled", fmt.Sprintf("%v", m.GetEnabled()))
			if m.GetAddress() != "" {
				t.add("MQTT", "Address", m.GetAddress())
			}
			if m.GetUsername() != "" {
				t.add("MQTT", "Username", m.GetUsername())
			}
			t.add("MQTT", "EncryptionEnabled", fmt.Sprintf("%v", m.GetEncryptionEnabled()))
			t.add("MQTT", "JsonEnabled", fmt.Sprintf("%v", m.GetJsonEnabled()))
		}
	case *pb.ModuleConfig_Serial:
		if s := v.Serial; s != nil {
			t.add("Serial", "Enabled", fmt.Sprintf("%v", s.GetEnabled()))
			t.add("Serial", "Baud", s.GetBaud().String())
			t.add("Serial", "Mode", s.GetMode().String())
		}
	case *pb.ModuleConfig_ExternalNotification:
		if e := v.ExternalNotification; e != nil {
			t.add("ExtNotify", "Enabled", fmt.Sprintf("%v", e.GetEnabled()))
			if e.GetOutputMs() > 0 {
				t.add("ExtNotify", "OutputMs", fmt.Sprintf("%d", e.GetOutputMs()))
			}
			t.add("ExtNotify", "AlertMessage", fmt.Sprintf("%v", e.GetAlertMessage()))
			t.add("ExtNotify", "AlertBell", fmt.Sprintf("%v", e.GetAlertBell()))
		}
	case *pb.ModuleConfig_StoreForward:
		if s := v.StoreForward; s != nil {
			t.add("StoreForward", "Enabled", fmt.Sprintf("%v", s.GetEnabled()))
			t.add("StoreForward", "Heartbeat", fmt.Sprintf("%v", s.GetHeartbeat()))
			if s.GetRecords() > 0 {
				t.add("StoreForward", "Records", fmt.Sprintf("%d", s.GetRecords()))
			}
			if s.GetHistoryReturnMax() > 0 {
				t.add("StoreForward", "HistoryReturnMax", fmt.Sprintf("%d", s.GetHistoryReturnMax()))
			}
			if s.GetHistoryReturnWindow() > 0 {
				t.add("StoreForward", "HistoryReturnWindow", fmt.Sprintf("%d", s.GetHistoryReturnWindow()))
			}
		}
	case *pb.ModuleConfig_RangeTest:
		if r := v.RangeTest; r != nil {
			t.add("RangeTest", "Enabled", fmt.Sprintf("%v", r.GetEnabled()))
			if r.GetSender() > 0 {
				t.add("RangeTest", "Sender", fmt.Sprintf("%d", r.GetSender()))
			}
		}
	case *pb.ModuleConfig_Telemetry:
		if te := v.Telemetry; te != nil {
			if te.GetDeviceUpdateInterval() > 0 {
				t.add("Telemetry", "DeviceUpdateInterval", fmt.Sprintf("%d", te.GetDeviceUpdateInterval()))
			}
			if te.GetEnvironmentUpdateInterval() > 0 {
				t.add("Telemetry", "EnvUpdateInterval", fmt.Sprintf("%d", te.GetEnvironmentUpdateInterval()))
			}
			t.add("Telemetry", "EnvMeasurementEnabled", fmt.Sprintf("%v", te.GetEnvironmentMeasurementEnabled()))
			t.add("Telemetry", "EnvScreenEnabled", fmt.Sprintf("%v", te.GetEnvironmentScreenEnabled()))
		}
	case *pb.ModuleConfig_CannedMessage:
		if c := v.CannedMessage; c != nil {
			t.add("CannedMsg", "Enabled", fmt.Sprintf("%v", c.GetEnabled()))
			if c.GetAllowInputSource() != "" {
				t.add("CannedMsg", "InputSource", c.GetAllowInputSource())
			}
		}
	case *pb.ModuleConfig_Audio:
		if a := v.Audio; a != nil {
			t.add("Audio", "Codec2Enabled", fmt.Sprintf("%v", a.GetCodec2Enabled()))
		}
	case *pb.ModuleConfig_NeighborInfo:
		if n := v.NeighborInfo; n != nil {
			t.add("NeighborInfo", "Enabled", fmt.Sprintf("%v", n.GetEnabled()))
			if n.GetUpdateInterval() > 0 {
				t.add("NeighborInfo", "UpdateInterval", fmt.Sprintf("%d", n.GetUpdateInterval()))
			}
		}
	case *pb.ModuleConfig_DetectionSensor:
		if d := v.DetectionSensor; d != nil {
			t.add("DetectSensor", "Enabled", fmt.Sprintf("%v", d.GetEnabled()))
			if d.GetName() != "" {
				t.add("DetectSensor", "Name", d.GetName())
			}
		}
	case *pb.ModuleConfig_Paxcounter:
		if p := v.Paxcounter; p != nil {
			t.add("Paxcounter", "Enabled", fmt.Sprintf("%v", p.GetEnabled()))
		}
	}
}

func (t *configTab) view(m *Model, height int) string {
	if len(t.entries) == 0 {
		msg := "  No config loaded. Press r to request from device."
		if t.received > 0 {
			msg = fmt.Sprintf("  Received %d packets but no fields parsed.\n  Types: %s\n  Press r to retry.",
				t.received, strings.Join(t.types, ", "))
		}
		return lipgloss.NewStyle().Foreground(ColorDim).Render(msg)
	}

	listHeight := height - 1
	if listHeight < 1 {
		listHeight = 1
	}

	hdr := Header.Render(fmt.Sprintf("  %-14s  %-28s  %s    (r:reload)", "SECTION", "KEY", "VALUE"))

	startIdx := 0
	if len(t.entries) > listHeight {
		halfView := listHeight / 2
		startIdx = t.cursor - halfView
		if startIdx < 0 {
			startIdx = 0
		}
		if startIdx > len(t.entries)-listHeight {
			startIdx = len(t.entries) - listHeight
		}
	}

	endIdx := startIdx + listHeight
	if endIdx > len(t.entries) {
		endIdx = len(t.entries)
	}

	var lines []string
	lines = append(lines, hdr)

	lastSection := ""
	for i := startIdx; i < endIdx; i++ {
		e := t.entries[i]
		section := e.section
		if section == lastSection {
			section = ""
		} else {
			lastSection = section
		}

		sectionStr := lipgloss.NewStyle().Foreground(ColorPurple).Bold(true).Width(14).Render(section)
		keyStr := lipgloss.NewStyle().Foreground(ColorCyan).Width(28).Render(e.key)
		valStr := lipgloss.NewStyle().Foreground(ColorWhite).Render(e.value)

		line := fmt.Sprintf("  %s  %s  %s", sectionStr, keyStr, valStr)

		if i == t.cursor {
			line = Selected.Width(m.width).Render(line)
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}
