package store

import "database/sql"

type Packet struct {
	ID        int64
	PacketID  int64
	FromNode  int64
	ToNode    int64
	Channel   int64
	Portnum   sql.NullInt64
	Timestamp int64
	RxTime    sql.NullInt64
	RxSNR     sql.NullFloat64
	RxRSSI    sql.NullInt64
	Raw       []byte
}

func (d *DB) InsertPacket(p Packet) error {
	_, err := d.db.Exec(`
		INSERT INTO packets (packet_id, from_node, to_node, channel, portnum, timestamp, rx_time, rx_snr, rx_rssi, raw)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.PacketID, p.FromNode, p.ToNode, p.Channel, p.Portnum,
		p.Timestamp, p.RxTime, p.RxSNR, p.RxRSSI, p.Raw)
	if err != nil {
		return err
	}

	d.prunePackets()
	return nil
}

func (d *DB) GetPackets(limit int) ([]Packet, error) {
	rows, err := d.db.Query("SELECT id, packet_id, from_node, to_node, channel, portnum, timestamp, rx_time, rx_snr, rx_rssi, raw FROM packets ORDER BY timestamp DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var packets []Packet
	for rows.Next() {
		var p Packet
		err := rows.Scan(&p.ID, &p.PacketID, &p.FromNode, &p.ToNode, &p.Channel,
			&p.Portnum, &p.Timestamp, &p.RxTime, &p.RxSNR, &p.RxRSSI, &p.Raw)
		if err != nil {
			return nil, err
		}
		packets = append(packets, p)
	}
	// Reverse to chronological order
	for i, j := 0, len(packets)-1; i < j; i, j = i+1, j-1 {
		packets[i], packets[j] = packets[j], packets[i]
	}
	return packets, nil
}

func (d *DB) prunePackets() {
	if d.packetLimit <= 0 {
		return
	}
	var count int
	d.db.QueryRow("SELECT COUNT(*) FROM packets").Scan(&count)
	if count > d.packetLimit {
		toDelete := count - d.packetLimit
		d.db.Exec("DELETE FROM packets WHERE id IN (SELECT id FROM packets ORDER BY timestamp ASC LIMIT ?)", toDelete)
	}
}

func (d *DB) GetPacketCount() int {
	var count int
	d.db.QueryRow("SELECT COUNT(*) FROM packets").Scan(&count)
	return count
}
