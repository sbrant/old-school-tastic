package bbs

import (
	"fmt"
	"strings"
	"time"

	"meshtastic-cli/internal/proto"
	"meshtastic-cli/internal/store"
)

type LogEntry struct {
	Timestamp time.Time
	FromNode  uint32
	FromName  string
	Command   string
	Response  string
}

type BBS struct {
	db        *store.DB
	startTime time.Time
	log       []LogEntry
	maxLog    int
}

func New(db *store.DB) *BBS {
	return &BBS{
		db:        db,
		startTime: time.Now(),
		maxLog:    500,
	}
}

func (b *BBS) Log() []LogEntry {
	return b.log
}

func (b *BBS) addLog(from uint32, fromName, cmd, response string) {
	entry := LogEntry{
		Timestamp: time.Now(),
		FromNode:  from,
		FromName:  fromName,
		Command:   cmd,
		Response:  response,
	}
	b.log = append(b.log, entry)
	if len(b.log) > b.maxLog {
		b.log = b.log[len(b.log)-b.maxLog:]
	}
}

// Handle processes an incoming text message and returns a response (or empty string for no response)
func (b *BBS) Handle(fromNode uint32, text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	fromName := b.db.GetNodeName(fromNode)
	if fromName == "" {
		fromName = proto.NodeIDStr(fromNode)
	}

	parts := strings.Fields(text)
	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	var response string

	switch cmd {
	case "help", "?":
		response = b.cmdHelp()
	case "ping":
		response = "pong"
	case "time":
		response = time.Now().Format("2006-01-02 15:04:05 MST")
	case "uptime":
		response = b.cmdUptime()
	case "nodes":
		response = b.cmdNodes()
	case "info":
		response = b.cmdInfo(fromNode)
	case "mail":
		response = b.cmdMail(fromNode, args)
	default:
		response = fmt.Sprintf("Unknown cmd: %s. Send 'help' for commands.", cmd)
	}

	b.addLog(fromNode, fromName, text, response)
	return response
}

func (b *BBS) cmdHelp() string {
	return "BBS Commands:\n" +
		"help     - this message\n" +
		"ping     - pong\n" +
		"time     - current time\n" +
		"uptime   - BBS uptime\n" +
		"nodes    - list mesh nodes\n" +
		"info     - your node info\n" +
		"mail read    - read your mail\n" +
		"mail send <name> <msg>\n" +
		"mail list    - mailbox status"
}

func (b *BBS) cmdUptime() string {
	d := time.Since(b.startTime)
	if d < time.Hour {
		return fmt.Sprintf("Uptime: %dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("Uptime: %dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("Uptime: %dd %dh", days, hours)
}

func (b *BBS) cmdNodes() string {
	nodes, err := b.db.GetAllNodes()
	if err != nil || len(nodes) == 0 {
		return "No nodes known"
	}

	// Limit to 10 to fit in a mesh message
	max := 10
	if len(nodes) < max {
		max = len(nodes)
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Nodes (%d total):", len(nodes)))
	for _, n := range nodes[:max] {
		name := "?"
		if n.LongName.Valid && n.LongName.String != "" {
			name = n.LongName.String
		}
		lastSeen := ""
		if n.LastHeard.Valid && n.LastHeard.Int64 > 0 {
			ago := time.Since(time.Unix(n.LastHeard.Int64, 0))
			if ago < time.Hour {
				lastSeen = fmt.Sprintf("%dm", int(ago.Minutes()))
			} else if ago < 24*time.Hour {
				lastSeen = fmt.Sprintf("%dh", int(ago.Hours()))
			} else {
				lastSeen = fmt.Sprintf("%dd", int(ago.Hours())/24)
			}
		}
		lines = append(lines, fmt.Sprintf("  %s (%s)", name, lastSeen))
	}
	if len(nodes) > max {
		lines = append(lines, fmt.Sprintf("  ...and %d more", len(nodes)-max))
	}

	return strings.Join(lines, "\n")
}

func (b *BBS) cmdInfo(fromNode uint32) string {
	name := b.db.GetNodeName(fromNode)
	if name == "" {
		name = "unknown"
	}
	id := proto.NodeIDStr(fromNode)
	return fmt.Sprintf("You: %s (%s)", name, id)
}

func (b *BBS) cmdMail(fromNode uint32, args []string) string {
	if len(args) == 0 {
		return "Usage: mail read | mail send <name> <msg> | mail list"
	}

	subcmd := strings.ToLower(args[0])

	switch subcmd {
	case "read":
		return b.mailRead(fromNode)
	case "send":
		if len(args) < 3 {
			return "Usage: mail send <name> <message>"
		}
		return b.mailSend(fromNode, args[1], strings.Join(args[2:], " "))
	case "list":
		return b.mailList()
	default:
		return "Usage: mail read | mail send <name> <msg> | mail list"
	}
}

func (b *BBS) mailRead(fromNode uint32) string {
	msgs, err := b.db.GetUnreadMailFor(fromNode)
	if err != nil || len(msgs) == 0 {
		return "No new mail"
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("You have %d message(s):", len(msgs)))
	for _, m := range msgs {
		fromName := b.db.GetNodeName(m.FromNode)
		if fromName == "" {
			fromName = proto.NodeIDStr(m.FromNode)
		}
		ts := time.UnixMilli(m.Timestamp).Format("Jan 02 15:04")
		lines = append(lines, fmt.Sprintf("  [%s] %s: %s", ts, fromName, m.Text))
	}

	b.db.MarkMailRead(fromNode)

	// Truncate if too long for mesh
	result := strings.Join(lines, "\n")
	if len(result) > 200 {
		result = result[:200] + "..."
	}
	return result
}

func (b *BBS) mailSend(fromNode uint32, targetName string, text string) string {
	// Find target node by name
	nodes, err := b.db.GetAllNodes()
	if err != nil {
		return "Error looking up nodes"
	}

	targetName = strings.ToLower(targetName)
	var targetNum uint32
	var matchedName string

	for _, n := range nodes {
		if n.ShortName.Valid && strings.ToLower(n.ShortName.String) == targetName {
			targetNum = uint32(n.Num)
			matchedName = n.ShortName.String
			break
		}
		if n.LongName.Valid && strings.Contains(strings.ToLower(n.LongName.String), targetName) {
			targetNum = uint32(n.Num)
			matchedName = n.LongName.String
			break
		}
	}

	if targetNum == 0 {
		return fmt.Sprintf("Node %q not found", targetName)
	}

	b.db.InsertMail(store.MailMessage{
		FromNode:  fromNode,
		ToNode:    targetNum,
		Text:      text,
		Timestamp: time.Now().UnixMilli(),
	})

	return fmt.Sprintf("Mail sent to %s", matchedName)
}

func (b *BBS) mailList() string {
	total, unread := b.db.GetMailCount()
	return fmt.Sprintf("Mailbox: %d messages (%d unread)", total, unread)
}
