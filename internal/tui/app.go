package tui

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	pb "buf.build/gen/go/meshtastic/protobufs/protocolbuffers/go/meshtastic"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"meshtastic-cli/internal/logger"
	"meshtastic-cli/internal/pcap"
	"meshtastic-cli/internal/proto"
	serialpkg "meshtastic-cli/internal/serial"
	"meshtastic-cli/internal/bbs"
	"meshtastic-cli/internal/store"
)

type tab int

const (
	tabPackets tab = iota
	tabNodes
	tabChat
	tabDM
	tabConfig
	tabChannels
	tabBBS
)

var tabNames = []string{"Packets", "Nodes", "Chat", "DM", "Config", "Channels", "BBS"}

// Messages

type PacketMsg struct {
	Packet *proto.Packet
}

type TickMsg time.Time

func waitForPacket(ch <-chan []byte) tea.Cmd {
	return func() tea.Msg {
		frame, ok := <-ch
		if !ok {
			return nil
		}
		pkt, err := proto.DecodeFromRadio(frame)
		if err != nil {
			return nil
		}
		return PacketMsg{Packet: pkt}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Model

type Options struct {
	Bot        bool
	BBS        bool
	Fahrenheit bool
	PcapFile   string
}

type Model struct {
	activeTab   tab
	width       int
	height      int
	showHelp    bool
	connected   bool
	myNodeNum   uint32
	nodeCount   int
	packetCount int
	typing      bool // true when text input has focus
	showDetail  bool // node detail overlay

	conn       *serialpkg.Conn
	db         *store.DB
	channels   []string // channel names by index
	opts       Options
	pcap       *pcap.Writer
	bbs        *bbs.BBS
	fileClient *bbs.FileClient

	packets    *packetsTab
	nodes      *nodesTab
	chat       *chatTab
	dm         *dmTab
	config     *configTab
	channelsUI *channelsTab
	bbsTab     *bbsTab
}

func NewModel(conn *serialpkg.Conn, db *store.DB, opts Options) Model {
	var pw *pcap.Writer
	if opts.PcapFile != "" {
		var err error
		pw, err = pcap.NewWriter(opts.PcapFile)
		if err != nil {
			logger.Error("App", fmt.Sprintf("failed to open pcap file: %v", err))
		}
	}

	var bbsEngine *bbs.BBS
	if opts.BBS {
		bbsEngine = bbs.New(db)
	}

	fileClient := bbs.NewFileClient(bbs.DefaultSaveDir())

	return Model{
		activeTab:  tabPackets,
		conn:       conn,
		db:         db,
		channels:   make([]string, 8),
		opts:       opts,
		pcap:       pw,
		bbs:        bbsEngine,
		fileClient: fileClient,
		packets:    newPacketsTab(),
		nodes:      newNodesTab(),
		chat:       newChatTab(),
		dm:         newDMTab(),
		config:     newConfigTab(),
		channelsUI: newChannelsTab(),
		bbsTab:     newBBSTab(),
	}
}

type configRequestedMsg struct{}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		waitForPacket(m.conn.Packets),
		tickCmd(),
		tea.EnterAltScreen,
		// Delay config request so the packet consumer is running first
		tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
			return configRequestedMsg{}
		}),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.chat.input.Width = msg.Width - 6
		m.dm.input.Width = msg.Width - 6
		return m, nil

	case configRequestedMsg:
		m.conn.RequestConfig()
		return m, nil

	case TickMsg:
		nodes, err := m.db.GetAllNodes()
		if err == nil {
			m.nodeCount = len(nodes)
			m.nodes.nodes = nodes
		}
		m.packetCount = m.db.GetPacketCount()
		// Refresh chat/dm if active
		if m.activeTab == tabChat {
			m.chat.refresh(m.db)
		} else if m.activeTab == tabDM && m.dm.mode == dmModeChat {
			m.dm.refreshMessages(m.db, m.myNodeNum)
		}
		return m, tickCmd()

	case PacketMsg:
		if msg.Packet != nil {
			m.processPacket(msg.Packet)
		} else {
			m.connected = false
		}
		return m, waitForPacket(m.conn.Packets)

	case nil:
		m.connected = false
		return m, nil

	case tea.KeyMsg:
		// When typing, send keys to the text input
		if m.typing {
			return m.handleTyping(msg)
		}

		// Node detail overlay
		if m.showDetail {
			switch msg.String() {
			case "esc", "q":
				m.showDetail = false
			case "enter":
				// Open DM with selected node
				if m.nodes.cursor < len(m.nodes.nodes) {
					node := m.nodes.nodes[m.nodes.cursor]
					m.showDetail = false
					m.dm.activeNode = uint32(node.Num)
					m.dm.mode = dmModeChat
					m.dm.refreshMessages(m.db, m.myNodeNum)
					m.dm.input.Focus()
					m.typing = true
					m.activeTab = tabDM
				}
			}
			return m, nil
		}

		// Help overlay
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			if m.pcap != nil {
				m.pcap.Close()
			}
			return m, tea.Quit
		case "?":
			m.showHelp = true
			return m, nil
		case "1":
			m.activeTab = tabPackets
		case "2":
			m.activeTab = tabNodes
			m.nodes.refresh(m.db)
		case "3":
			m.activeTab = tabChat
			m.chat.refresh(m.db)
		case "4":
			m.activeTab = tabDM
			m.dm.refresh(m.db, m.myNodeNum)
		case "5":
			m.activeTab = tabConfig
		case "6":
			m.activeTab = tabChannels
			if m.myNodeNum != 0 {
				m.requestChannels()
			}
		case "7":
			m.activeTab = tabBBS

		// Re-request config
		case "r":
			if m.activeTab == tabConfig {
				m.requestConfig()
			} else if m.activeTab == tabChannels {
				m.requestChannels()
			}

		// Channel-specific keys
		case "n":
			if m.activeTab == tabChannels && !m.channelsUI.editing && m.myNodeNum != 0 {
				m.startChannelCreate()
			}
		case "d":
			if m.activeTab == tabChannels && !m.channelsUI.editing && m.myNodeNum != 0 {
				m.disableChannel()
			}

		// Enter key
		case "enter":
			switch m.activeTab {
			case tabNodes:
				if m.nodes.cursor < len(m.nodes.nodes) {
					m.showDetail = true
				}
			case tabChat:
				m.chat.input.Focus()
				m.typing = true
				return m, textinput.Blink
			case tabDM:
				if m.dm.mode == dmModeList {
					if m.dm.cursor < len(m.dm.conversations) {
						c := m.dm.conversations[m.dm.cursor]
						m.dm.activeNode = c.NodeNum
						m.dm.mode = dmModeChat
						m.dm.refreshMessages(m.db, m.myNodeNum)
						m.dm.input.Focus()
						m.typing = true
						return m, textinput.Blink
					}
				} else {
					m.dm.input.Focus()
					m.typing = true
					return m, textinput.Blink
				}
			case tabChannels:
				if !m.channelsUI.editing && m.channelsUI.cursor < len(m.channelsUI.channels) {
					m.startChannelEdit()
					return m, textinput.Blink
				}
			}

		// Escape
		case "esc":
			if m.activeTab == tabDM && m.dm.mode == dmModeChat {
				m.dm.mode = dmModeList
				m.dm.refresh(m.db, m.myNodeNum)
			} else if m.activeTab == tabChannels && m.channelsUI.editing {
				m.channelsUI.editing = false
				m.channelsUI.creating = false
				m.typing = false
				m.channelsUI.input.Blur()
			}

		// Vim navigation
		case "j", "down":
			m.moveCursor(1)
		case "k", "up":
			m.moveCursor(-1)
		case "g", "home":
			m.setCursor(0)
		case "G", "end":
			m.setCursorEnd()
		case "ctrl+d":
			m.moveCursor(m.height / 2)
		case "ctrl+u":
			m.moveCursor(-m.height / 2)
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) handleTyping(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.typing = false
		m.chat.input.Blur()
		m.dm.input.Blur()
		if m.activeTab == tabChannels {
			m.channelsUI.editing = false
			m.channelsUI.creating = false
			m.channelsUI.input.Blur()
		}
		return m, nil

	case "tab":
		// Cycle fields in channel editor
		if m.activeTab == tabChannels && m.channelsUI.editing {
			m.channelNextField()
			return m, nil
		}

	case "g":
		// Generate random PSK in channel editor
		if m.activeTab == tabChannels && m.channelsUI.editing && m.channelsUI.field == fieldPSK {
			psk := generatePSK()
			m.channelsUI.input.SetValue(hex.EncodeToString(psk))
			return m, nil
		}

	case "enter":
		switch m.activeTab {
		case tabChat:
			text := m.chat.input.Value()
			if text != "" && m.myNodeNum != 0 {
				m.sendMessage(text, 0xFFFFFFFF, uint32(m.chat.channel))
				m.chat.input.SetValue("")
				m.chat.refresh(m.db)
			}
		case tabDM:
			text := m.dm.input.Value()
			if text != "" && m.myNodeNum != 0 && m.dm.activeNode != 0 {
				m.sendMessage(text, m.dm.activeNode, 0)
				m.dm.input.SetValue("")
				m.dm.refreshMessages(m.db, m.myNodeNum)
			}
		case tabChannels:
			if m.channelsUI.editing {
				m.saveChannelEdit()
				return m, nil
			}
		}
		return m, nil
	}

	// Forward to the active text input
	var cmd tea.Cmd
	switch m.activeTab {
	case tabChat:
		m.chat.input, cmd = m.chat.input.Update(msg)
	case tabDM:
		m.dm.input, cmd = m.dm.input.Update(msg)
	case tabChannels:
		m.channelsUI.input, cmd = m.channelsUI.input.Update(msg)
	}
	return m, cmd
}

func (m *Model) requestConfig() {
	if m.myNodeNum == 0 {
		nonce := uint32(time.Now().UnixNano() & 0xFFFFFFFF)
		data, _ := proto.EncodeWantConfig(nonce)
		m.conn.Send(data)
		return
	}

	go func() {
		configTypes := []pb.AdminMessage_ConfigType{
			pb.AdminMessage_DEVICE_CONFIG,
			pb.AdminMessage_POSITION_CONFIG,
			pb.AdminMessage_POWER_CONFIG,
			pb.AdminMessage_NETWORK_CONFIG,
			pb.AdminMessage_DISPLAY_CONFIG,
			pb.AdminMessage_LORA_CONFIG,
			pb.AdminMessage_BLUETOOTH_CONFIG,
			pb.AdminMessage_SECURITY_CONFIG,
		}
		for _, ct := range configTypes {
			data, err := proto.EncodeAdminConfigRequest(m.myNodeNum, ct)
			if err == nil {
				m.conn.Send(data)
			}
			time.Sleep(100 * time.Millisecond)
		}

		moduleTypes := []pb.AdminMessage_ModuleConfigType{
			pb.AdminMessage_MQTT_CONFIG,
			pb.AdminMessage_SERIAL_CONFIG,
			pb.AdminMessage_TELEMETRY_CONFIG,
			pb.AdminMessage_STOREFORWARD_CONFIG,
			pb.AdminMessage_NEIGHBORINFO_CONFIG,
		}
		for _, mt := range moduleTypes {
			data, err := proto.EncodeAdminModuleConfigRequest(m.myNodeNum, mt)
			if err == nil {
				m.conn.Send(data)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
}

func (m *Model) sendMessage(text string, to uint32, channel uint32) {
	data, err := proto.EncodeTextMessage(m.myNodeNum, to, text, channel)
	if err != nil {
		return
	}
	if err := m.conn.Send(data); err != nil {
		return
	}

	now := time.Now().UnixMilli()
	m.db.InsertMessage(store.Message{
		PacketID:  0,
		FromNode:  int64(m.myNodeNum),
		ToNode:    int64(to),
		Channel:   int64(channel),
		Text:      text,
		Timestamp: now,
		Status:    "pending",
	})
}

func (m *Model) sendChunked(msgs []string, to uint32, channel uint32) {
	for i, msg := range msgs {
		m.sendMessage(msg, to, channel)
		if i < len(msgs)-1 {
			time.Sleep(2 * time.Second)
		}
	}
}

func (m *Model) requestChannels() {
	if m.myNodeNum == 0 {
		return
	}
	go func() {
		for i := uint32(0); i < 8; i++ {
			data, err := proto.EncodeAdminGetChannel(m.myNodeNum, i)
			if err == nil {
				m.conn.Send(data)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
}

func (m *Model) startChannelEdit() {
	ch := m.channelsUI.channels[m.channelsUI.cursor]
	m.channelsUI.editing = true
	m.channelsUI.creating = false
	m.channelsUI.editIdx = m.channelsUI.cursor
	m.channelsUI.field = fieldName
	m.channelsUI.input.SetValue(ch.name)
	m.channelsUI.input.Focus()
	m.typing = true
}

func (m *Model) startChannelCreate() {
	// Find first disabled slot
	idx := int32(-1)
	for i := int32(1); i < 8; i++ {
		found := false
		for _, ch := range m.channelsUI.channels {
			if ch.index == i && ch.role != pb.Channel_DISABLED {
				found = true
				break
			}
		}
		if !found {
			idx = i
			break
		}
	}
	if idx == -1 {
		m.channelsUI.message = "No available channel slots"
		return
	}

	// Add a placeholder
	m.channelsUI.channels = append(m.channelsUI.channels, channelInfo{
		index: idx,
		role:  pb.Channel_SECONDARY,
		psk:   generatePSK(),
	})
	m.channelsUI.cursor = len(m.channelsUI.channels) - 1
	m.channelsUI.editing = true
	m.channelsUI.creating = true
	m.channelsUI.editIdx = m.channelsUI.cursor
	m.channelsUI.field = fieldName
	m.channelsUI.input.SetValue("")
	m.channelsUI.input.Focus()
	m.typing = true
}

func (m *Model) channelNextField() {
	ch := &m.channelsUI.channels[m.channelsUI.editIdx]

	// Save current field value
	switch m.channelsUI.field {
	case fieldName:
		ch.name = m.channelsUI.input.Value()
		m.channelsUI.field = fieldPSK
		m.channelsUI.input.SetValue(pskDisplayStr(ch.psk))
	case fieldPSK:
		// Parse PSK from input
		val := m.channelsUI.input.Value()
		if val == "" || val == "(none)" {
			ch.psk = nil
		} else if val == "default" {
			ch.psk = []byte{1}
		} else if decoded, err := hex.DecodeString(val); err == nil {
			ch.psk = decoded
		}
		m.channelsUI.field = fieldRole
		m.channelsUI.input.SetValue(ch.role.String())
	case fieldRole:
		val := strings.ToUpper(m.channelsUI.input.Value())
		if strings.Contains(val, "SEC") {
			ch.role = pb.Channel_SECONDARY
		} else if strings.Contains(val, "PRI") {
			ch.role = pb.Channel_PRIMARY
		} else if strings.Contains(val, "DIS") {
			ch.role = pb.Channel_DISABLED
		}
		m.channelsUI.field = fieldName
		m.channelsUI.input.SetValue(ch.name)
	}
}

func (m *Model) saveChannelEdit() {
	ch := &m.channelsUI.channels[m.channelsUI.editIdx]

	// Save current field
	switch m.channelsUI.field {
	case fieldName:
		ch.name = m.channelsUI.input.Value()
	case fieldPSK:
		val := m.channelsUI.input.Value()
		if val == "" || val == "(none)" {
			ch.psk = nil
		} else if val == "default" {
			ch.psk = []byte{1}
		} else if decoded, err := hex.DecodeString(val); err == nil {
			ch.psk = decoded
		}
	case fieldRole:
		val := strings.ToUpper(m.channelsUI.input.Value())
		if strings.Contains(val, "SEC") {
			ch.role = pb.Channel_SECONDARY
		} else if strings.Contains(val, "PRI") {
			ch.role = pb.Channel_PRIMARY
		} else if strings.Contains(val, "DIS") {
			ch.role = pb.Channel_DISABLED
		}
	}

	// Send to device
	data, err := buildSetChannel(m.myNodeNum, ch.index, ch.name, ch.psk, ch.role)
	if err == nil {
		m.conn.Send(data)
		m.channelsUI.message = fmt.Sprintf("Channel %d updated", ch.index)
	} else {
		m.channelsUI.message = "Error: " + err.Error()
	}

	m.channelsUI.editing = false
	m.channelsUI.creating = false
	m.channelsUI.input.Blur()
	m.typing = false

	// Also update the local channel names cache
	if int(ch.index) < len(m.channels) {
		m.channels[ch.index] = ch.name
	}
}

func (m *Model) disableChannel() {
	if m.channelsUI.cursor >= len(m.channelsUI.channels) {
		return
	}
	ch := m.channelsUI.channels[m.channelsUI.cursor]
	if ch.role == pb.Channel_PRIMARY {
		m.channelsUI.message = "Cannot disable primary channel"
		return
	}
	data, err := buildSetChannel(m.myNodeNum, ch.index, "", nil, pb.Channel_DISABLED)
	if err == nil {
		m.conn.Send(data)
		m.channelsUI.channels[m.channelsUI.cursor].role = pb.Channel_DISABLED
		m.channelsUI.message = fmt.Sprintf("Channel %d disabled", ch.index)
	}
}

func (m *Model) moveCursor(delta int) {
	switch m.activeTab {
	case tabPackets:
		m.packets.cursor += delta
		max := len(m.packets.packets) - 1
		if m.packets.cursor < 0 {
			m.packets.cursor = 0
		}
		if m.packets.cursor > max {
			m.packets.cursor = max
		}
	case tabNodes:
		m.nodes.cursor += delta
		max := len(m.nodes.nodes) - 1
		if m.nodes.cursor < 0 {
			m.nodes.cursor = 0
		}
		if m.nodes.cursor > max {
			m.nodes.cursor = max
		}
	case tabConfig:
		m.config.cursor += delta
		max := len(m.config.entries) - 1
		if m.config.cursor < 0 {
			m.config.cursor = 0
		}
		if m.config.cursor > max {
			m.config.cursor = max
		}
	case tabChannels:
		if !m.channelsUI.editing {
			m.channelsUI.cursor += delta
			max := len(m.channelsUI.channels) - 1
			if m.channelsUI.cursor < 0 {
				m.channelsUI.cursor = 0
			}
			if m.channelsUI.cursor > max {
				m.channelsUI.cursor = max
			}
		}
	case tabDM:
		if m.dm.mode == dmModeList {
			m.dm.cursor += delta
			max := len(m.dm.conversations) - 1
			if m.dm.cursor < 0 {
				m.dm.cursor = 0
			}
			if m.dm.cursor > max {
				m.dm.cursor = max
			}
		}
	}
}

func (m *Model) setCursor(pos int) {
	switch m.activeTab {
	case tabPackets:
		m.packets.cursor = pos
	case tabNodes:
		m.nodes.cursor = pos
	case tabConfig:
		m.config.cursor = pos
	case tabDM:
		if m.dm.mode == dmModeList {
			m.dm.cursor = pos
		}
	}
}

func (m *Model) setCursorEnd() {
	switch m.activeTab {
	case tabPackets:
		if len(m.packets.packets) > 0 {
			m.packets.cursor = len(m.packets.packets) - 1
		}
	case tabNodes:
		if len(m.nodes.nodes) > 0 {
			m.nodes.cursor = len(m.nodes.nodes) - 1
		}
	case tabConfig:
		if len(m.config.entries) > 0 {
			m.config.cursor = len(m.config.entries) - 1
		}
	case tabDM:
		if m.dm.mode == dmModeList && len(m.dm.conversations) > 0 {
			m.dm.cursor = len(m.dm.conversations) - 1
		}
	}
}

func (m *Model) processPacket(pkt *proto.Packet) {
	fr := pkt.FromRadio
	now := time.Now().UnixMilli()

	m.connected = true
	m.packets.addPacket(pkt)

	// Write to pcap if enabled
	if m.pcap != nil && pkt.Raw != nil {
		m.pcap.WritePacket(pkt.Raw, time.Now())
	}

	switch v := fr.PayloadVariant.(type) {
	case *pb.FromRadio_MyInfo:
		if m.myNodeNum == 0 {
			m.myNodeNum = v.MyInfo.GetMyNodeNum()
			// Now that we know our node num, request full config via admin
			m.requestConfig()
		}

	case *pb.FromRadio_Channel:
		ch := v.Channel
		if int(ch.GetIndex()) < len(m.channels) && ch.GetSettings() != nil {
			m.channels[ch.GetIndex()] = ch.GetSettings().GetName()
		}
		m.channelsUI.addChannel(ch)

	case *pb.FromRadio_Config:
		m.config.addConfig(v.Config)
		logger.Info("Config", fmt.Sprintf("received config: %T entries=%d", v.Config.PayloadVariant, len(m.config.entries)))

	case *pb.FromRadio_ModuleConfig:
		m.config.addModuleConfig(v.ModuleConfig)
		logger.Info("Config", fmt.Sprintf("received module config: %T entries=%d", v.ModuleConfig.PayloadVariant, len(m.config.entries)))

	case *pb.FromRadio_NodeInfo:
		ni := v.NodeInfo
		node := store.Node{Num: int64(ni.GetNum())}
		if ni.GetUser() != nil {
			node.UserID = nullStr(ni.GetUser().GetId())
			node.LongName = nullStr(ni.GetUser().GetLongName())
			node.ShortName = nullStr(ni.GetUser().GetShortName())
			node.HwModel = sql.NullInt64{Int64: int64(ni.GetUser().GetHwModel()), Valid: true}
			node.Role = sql.NullInt64{Int64: int64(ni.GetUser().GetRole()), Valid: true}
		}
		if ni.GetPosition() != nil {
			node.LatitudeI = nullInt64(int64(ni.GetPosition().GetLatitudeI()))
			node.LongitudeI = nullInt64(int64(ni.GetPosition().GetLongitudeI()))
			node.Altitude = nullInt64(int64(ni.GetPosition().GetAltitude()))
		}
		if ni.GetSnr() != 0 {
			node.SNR = sql.NullFloat64{Float64: float64(ni.GetSnr()), Valid: true}
		}
		if ni.GetLastHeard() != 0 {
			node.LastHeard = nullInt64(int64(ni.GetLastHeard()))
		}
		if ni.GetDeviceMetrics() != nil {
			dm := ni.GetDeviceMetrics()
			if dm.GetBatteryLevel() > 0 {
				node.BatteryLevel = nullInt64(int64(dm.GetBatteryLevel()))
			}
			if dm.GetVoltage() > 0 {
				node.Voltage = sql.NullFloat64{Float64: float64(dm.GetVoltage()), Valid: true}
			}
			if dm.GetChannelUtilization() > 0 {
				node.ChannelUtilization = sql.NullFloat64{Float64: float64(dm.GetChannelUtilization()), Valid: true}
			}
			if dm.GetAirUtilTx() > 0 {
				node.AirUtilTx = sql.NullFloat64{Float64: float64(dm.GetAirUtilTx()), Valid: true}
			}
		}
		if ni.GetChannel() != 0 {
			node.Channel = nullInt64(int64(ni.GetChannel()))
		}
		m.db.UpsertNode(node)

	case *pb.FromRadio_Packet:
		mp := v.Packet
		d := mp.GetDecoded()
		if d == nil {
			return
		}

		dbPkt := store.Packet{
			PacketID:  int64(mp.GetId()),
			FromNode:  int64(mp.GetFrom()),
			ToNode:    int64(mp.GetTo()),
			Channel:   int64(mp.GetChannel()),
			Timestamp: now,
			Raw:       pkt.Raw,
			Portnum:   sql.NullInt64{Int64: int64(d.GetPortnum()), Valid: true},
		}
		if mp.GetRxTime() != 0 {
			dbPkt.RxTime = nullInt64(int64(mp.GetRxTime()))
		}
		if mp.GetRxSnr() != 0 {
			dbPkt.RxSNR = sql.NullFloat64{Float64: float64(mp.GetRxSnr()), Valid: true}
		}
		if mp.GetRxRssi() != 0 {
			dbPkt.RxRSSI = nullInt64(int64(mp.GetRxRssi()))
		}
		m.db.InsertPacket(dbPkt)

		nodeUpdate := store.Node{Num: int64(mp.GetFrom())}
		if mp.GetRxSnr() != 0 {
			nodeUpdate.SNR = sql.NullFloat64{Float64: float64(mp.GetRxSnr()), Valid: true}
		}
		nodeUpdate.LastHeard = nullInt64(now / 1000)

		switch payload := pkt.Payload.(type) {
		case string:
			m.db.InsertMessage(store.Message{
				PacketID:  int64(mp.GetId()),
				FromNode:  int64(mp.GetFrom()),
				ToNode:    int64(mp.GetTo()),
				Channel:   int64(mp.GetChannel()),
				Text:      payload,
				Timestamp: now,
				RxTime:    dbPkt.RxTime,
				RxSNR:     dbPkt.RxSNR,
				RxRSSI:    dbPkt.RxRSSI,
				HopLimit:  nullInt64(int64(mp.GetHopLimit())),
				Status:    "received",
			})
			// File client — intercept incoming file chunks
			if m.fileClient != nil {
				m.fileClient.HandleMessage(mp.GetFrom(), payload)
				m.fileClient.CleanStale()
			}
			// BBS auto-reply — only on encrypted channels (not ch 0), not broadcast
			if m.bbs != nil && m.myNodeNum != 0 && mp.GetFrom() != m.myNodeNum &&
				mp.GetChannel() != 0 && mp.GetTo() != 0xFFFFFFFF {
				replies := m.bbs.Handle(mp.GetFrom(), payload)
				if len(replies) > 0 {
					go m.sendChunked(replies, mp.GetFrom(), mp.GetChannel())
				}
			} else if m.opts.Bot && m.myNodeNum != 0 && mp.GetFrom() != m.myNodeNum {
				// Simple bot mode (legacy)
				lower := strings.ToLower(strings.TrimSpace(payload))
				if lower == "ping" || lower == "test" {
					reply := "pong"
					if lower == "test" {
						reply = "ack"
					}
					m.sendMessage(reply, mp.GetFrom(), mp.GetChannel())
				}
			}
		case *pb.User:
			nodeUpdate.UserID = nullStr(payload.GetId())
			nodeUpdate.LongName = nullStr(payload.GetLongName())
			nodeUpdate.ShortName = nullStr(payload.GetShortName())
			nodeUpdate.HwModel = sql.NullInt64{Int64: int64(payload.GetHwModel()), Valid: true}
			nodeUpdate.Role = sql.NullInt64{Int64: int64(payload.GetRole()), Valid: true}
		case *pb.Position:
			if payload.GetLatitudeI() != 0 || payload.GetLongitudeI() != 0 {
				nodeUpdate.LatitudeI = nullInt64(int64(payload.GetLatitudeI()))
				nodeUpdate.LongitudeI = nullInt64(int64(payload.GetLongitudeI()))
			}
			if payload.GetAltitude() != 0 {
				nodeUpdate.Altitude = nullInt64(int64(payload.GetAltitude()))
			}
		case *pb.Telemetry:
			if dm := payload.GetDeviceMetrics(); dm != nil {
				if dm.GetBatteryLevel() > 0 {
					nodeUpdate.BatteryLevel = nullInt64(int64(dm.GetBatteryLevel()))
				}
				if dm.GetVoltage() > 0 {
					nodeUpdate.Voltage = sql.NullFloat64{Float64: float64(dm.GetVoltage()), Valid: true}
				}
				if dm.GetChannelUtilization() > 0 {
					nodeUpdate.ChannelUtilization = sql.NullFloat64{Float64: float64(dm.GetChannelUtilization()), Valid: true}
				}
				if dm.GetAirUtilTx() > 0 {
					nodeUpdate.AirUtilTx = sql.NullFloat64{Float64: float64(dm.GetAirUtilTx()), Valid: true}
				}
			}
		case *pb.AdminMessage:
			if cfg := payload.GetGetConfigResponse(); cfg != nil {
				m.config.addConfig(cfg)
			}
			if mcfg := payload.GetGetModuleConfigResponse(); mcfg != nil {
				m.config.addModuleConfig(mcfg)
			}
			if ch := payload.GetGetChannelResponse(); ch != nil {
				m.channelsUI.addChannel(ch)
			}
		}

		m.db.UpsertNode(nodeUpdate)
	}
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	// Overlays
	if m.showHelp {
		return renderHelp(m.width, m.height)
	}
	if m.showDetail && m.activeTab == tabNodes && m.nodes.cursor < len(m.nodes.nodes) {
		return renderNodeDetail(m.nodes.nodes[m.nodes.cursor], m.width, m.height)
	}

	// Tab bar
	var tabs []string
	for i, name := range tabNames {
		label := fmt.Sprintf(" %d:%s ", i+1, name)
		if tab(i) == m.activeTab {
			tabs = append(tabs, TabActive.Render(label))
		} else {
			tabs = append(tabs, TabInactive.Render(label))
		}
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	tabBar = lipgloss.NewStyle().Width(m.width).Background(lipgloss.Color("#161b22")).Render(tabBar)

	// Available lines for content: total height minus tab bar (1) and status bar (1)
	contentHeight := m.height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Content — each tab gets the exact height to fill
	var content string
	switch m.activeTab {
	case tabPackets:
		content = m.packets.view(&m, contentHeight)
	case tabNodes:
		content = m.nodes.view(&m, contentHeight)
	case tabChat:
		content = m.chat.view(&m, contentHeight)
	case tabDM:
		content = m.dm.view(&m, contentHeight)
	case tabConfig:
		content = m.config.view(&m, contentHeight)
	case tabChannels:
		content = m.channelsUI.view(&m, contentHeight)
	case tabBBS:
		content = m.bbsTab.view(&m, contentHeight)
	}

	// Hard clamp content to exactly contentHeight lines
	contentLines := strings.Split(content, "\n")
	if len(contentLines) > contentHeight {
		contentLines = contentLines[:contentHeight]
	}
	for len(contentLines) < contentHeight {
		contentLines = append(contentLines, "")
	}

	statusBar := renderStatusBar(&m)

	// Assemble final output and enforce total height
	allLines := make([]string, 0, m.height)
	allLines = append(allLines, tabBar)
	allLines = append(allLines, contentLines...)
	allLines = append(allLines, statusBar)

	// Final safety clamp — never exceed terminal height
	if len(allLines) > m.height {
		allLines = allLines[:m.height]
	}

	return strings.Join(allLines, "\n")
}

func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullInt64(i int64) sql.NullInt64 {
	return sql.NullInt64{Int64: i, Valid: true}
}
