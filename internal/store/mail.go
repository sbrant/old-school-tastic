package store

import "database/sql"

type MailMessage struct {
	ID        int64
	FromNode  uint32
	ToNode    uint32
	Text      string
	Timestamp int64
	Read      bool
}

func (d *DB) InsertMail(m MailMessage) error {
	_, err := d.db.Exec(`
		INSERT INTO mail (from_node, to_node, text, timestamp, read)
		VALUES (?, ?, ?, ?, 0)`,
		m.FromNode, m.ToNode, m.Text, m.Timestamp)
	return err
}

func (d *DB) GetMailFor(nodeNum uint32) ([]MailMessage, error) {
	rows, err := d.db.Query(`
		SELECT id, from_node, to_node, text, timestamp, read
		FROM mail WHERE to_node = ? ORDER BY timestamp ASC`, nodeNum)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMail(rows)
}

func (d *DB) GetUnreadMailFor(nodeNum uint32) ([]MailMessage, error) {
	rows, err := d.db.Query(`
		SELECT id, from_node, to_node, text, timestamp, read
		FROM mail WHERE to_node = ? AND read = 0 ORDER BY timestamp ASC`, nodeNum)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMail(rows)
}

func (d *DB) MarkMailRead(nodeNum uint32) error {
	_, err := d.db.Exec("UPDATE mail SET read = 1 WHERE to_node = ?", nodeNum)
	return err
}

func (d *DB) GetMailCount() (total int, unread int) {
	d.db.QueryRow("SELECT COUNT(*) FROM mail").Scan(&total)
	d.db.QueryRow("SELECT COUNT(*) FROM mail WHERE read = 0").Scan(&unread)
	return
}

func scanMail(rows *sql.Rows) ([]MailMessage, error) {
	var msgs []MailMessage
	for rows.Next() {
		var m MailMessage
		err := rows.Scan(&m.ID, &m.FromNode, &m.ToNode, &m.Text, &m.Timestamp, &m.Read)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}
