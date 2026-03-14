package store

import "database/sql"

const broadcastAddr = 0xFFFFFFFF

type Message struct {
	ID        int64
	PacketID  int64
	FromNode  int64
	ToNode    int64
	Channel   int64
	Text      string
	Timestamp int64
	RxTime    sql.NullInt64
	RxSNR     sql.NullFloat64
	RxRSSI    sql.NullInt64
	HopLimit  sql.NullInt64
	Status    string
	ReplyID   sql.NullInt64
	ErrorReason sql.NullString
}

func (d *DB) InsertMessage(m Message) error {
	if m.Status == "" {
		m.Status = "received"
	}
	_, err := d.db.Exec(`
		INSERT INTO messages (packet_id, from_node, to_node, channel, text, timestamp, rx_time, rx_snr, rx_rssi, hop_limit, status, reply_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.PacketID, m.FromNode, m.ToNode, m.Channel, m.Text, m.Timestamp,
		m.RxTime, m.RxSNR, m.RxRSSI, m.HopLimit, m.Status, m.ReplyID)
	return err
}

func (d *DB) UpdateMessageStatus(packetID int64, status string, errorReason string) error {
	if errorReason != "" {
		_, err := d.db.Exec("UPDATE messages SET status = ?, error_reason = ? WHERE packet_id = ?", status, errorReason, packetID)
		return err
	}
	_, err := d.db.Exec("UPDATE messages SET status = ? WHERE packet_id = ?", status, packetID)
	return err
}

func (d *DB) GetMessages(channel int, limit int) ([]Message, error) {
	rows, err := d.db.Query(`
		SELECT id, packet_id, from_node, to_node, channel, text, timestamp, rx_time, rx_snr, rx_rssi, hop_limit, status, reply_id, error_reason
		FROM messages WHERE channel = ? AND to_node = ? ORDER BY timestamp DESC LIMIT ?`,
		channel, broadcastAddr, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMessages(rows)
}

func (d *DB) GetDMMessages(myNode, otherNode uint32, limit int) ([]Message, error) {
	rows, err := d.db.Query(`
		SELECT id, packet_id, from_node, to_node, channel, text, timestamp, rx_time, rx_snr, rx_rssi, hop_limit, status, reply_id, error_reason
		FROM messages
		WHERE to_node != ?
			AND ((from_node = ? AND to_node = ?) OR (from_node = ? AND to_node = ?))
		ORDER BY timestamp DESC LIMIT ?`,
		broadcastAddr, myNode, otherNode, otherNode, myNode, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMessages(rows)
}

type DMConversation struct {
	NodeNum       uint32
	LastMessage   string
	LastTimestamp  int64
	UnreadCount   int
}

func (d *DB) GetDMConversations(myNode uint32) ([]DMConversation, error) {
	rows, err := d.db.Query(`
		SELECT other_node, last_message, last_timestamp, unread_count FROM (
			SELECT
				CASE WHEN from_node = ? THEN to_node ELSE from_node END as other_node,
				MAX(timestamp) as last_timestamp,
				SUM(CASE WHEN from_node != ? AND status = 'received' THEN 1 ELSE 0 END) as unread_count
			FROM messages
			WHERE to_node != ? AND (from_node = ? OR to_node = ?)
			GROUP BY other_node
		) t
		LEFT JOIN LATERAL (
			SELECT text as last_message FROM messages m2
			WHERE m2.to_node != ?
				AND ((m2.from_node = ? AND m2.to_node = t.other_node) OR (m2.from_node = t.other_node AND m2.to_node = ?))
			ORDER BY m2.timestamp DESC LIMIT 1
		) ON true
		ORDER BY last_timestamp DESC`,
		myNode, myNode, broadcastAddr, myNode, myNode,
		broadcastAddr, myNode, myNode)
	if err != nil {
		// Fallback: simpler query without LATERAL (SQLite may not support it)
		return d.getDMConversationsSimple(myNode)
	}
	defer rows.Close()

	var convos []DMConversation
	for rows.Next() {
		var c DMConversation
		var lastMsg sql.NullString
		if err := rows.Scan(&c.NodeNum, &lastMsg, &c.LastTimestamp, &c.UnreadCount); err != nil {
			return nil, err
		}
		if lastMsg.Valid {
			c.LastMessage = lastMsg.String
		}
		convos = append(convos, c)
	}
	return convos, nil
}

func (d *DB) getDMConversationsSimple(myNode uint32) ([]DMConversation, error) {
	rows, err := d.db.Query(`
		SELECT
			CASE WHEN from_node = ? THEN to_node ELSE from_node END as other_node,
			MAX(timestamp) as last_timestamp,
			SUM(CASE WHEN from_node != ? AND status = 'received' THEN 1 ELSE 0 END) as unread_count
		FROM messages
		WHERE to_node != ? AND (from_node = ? OR to_node = ?)
		GROUP BY other_node
		ORDER BY last_timestamp DESC`,
		myNode, myNode, broadcastAddr, myNode, myNode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convos []DMConversation
	for rows.Next() {
		var c DMConversation
		if err := rows.Scan(&c.NodeNum, &c.LastTimestamp, &c.UnreadCount); err != nil {
			return nil, err
		}
		convos = append(convos, c)
	}
	return convos, nil
}

func scanMessages(rows *sql.Rows) ([]Message, error) {
	var msgs []Message
	for rows.Next() {
		var m Message
		err := rows.Scan(&m.ID, &m.PacketID, &m.FromNode, &m.ToNode, &m.Channel,
			&m.Text, &m.Timestamp, &m.RxTime, &m.RxSNR, &m.RxRSSI,
			&m.HopLimit, &m.Status, &m.ReplyID, &m.ErrorReason)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	// Reverse to get chronological order
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}
