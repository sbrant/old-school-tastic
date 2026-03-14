package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	db             *sql.DB
	session        string
	packetLimit    int
}

func dbDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "meshtastic-cli")
}

func DbPath(session string) string {
	return filepath.Join(dbDir(), session+".db")
}

func Open(session string, packetLimit int) (*DB, error) {
	dir := dbDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	path := filepath.Join(dir, session+".db")
	sqldb, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	sqldb.Exec("PRAGMA journal_mode = WAL")
	sqldb.Exec("PRAGMA busy_timeout = 5000")

	d := &DB{db: sqldb, session: session, packetLimit: packetLimit}
	if err := d.migrate(); err != nil {
		sqldb.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return d, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) migrate() error {
	d.db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER PRIMARY KEY,
		applied_at INTEGER NOT NULL
	)`)

	var version int
	row := d.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version")
	row.Scan(&version)

	migrations := []string{
		// v1: initial schema
		`CREATE TABLE IF NOT EXISTS nodes (
			num INTEGER PRIMARY KEY,
			user_id TEXT,
			long_name TEXT,
			short_name TEXT,
			hw_model INTEGER,
			role INTEGER,
			latitude_i INTEGER,
			longitude_i INTEGER,
			altitude INTEGER,
			snr REAL,
			last_heard INTEGER,
			battery_level INTEGER,
			voltage REAL,
			channel_utilization REAL,
			air_util_tx REAL,
			channel INTEGER,
			via_mqtt INTEGER,
			hops_away INTEGER,
			is_favorite INTEGER,
			public_key BLOB,
			updated_at INTEGER
		);
		CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			packet_id INTEGER,
			from_node INTEGER,
			to_node INTEGER,
			channel INTEGER,
			text TEXT,
			timestamp INTEGER,
			rx_time INTEGER,
			rx_snr REAL,
			rx_rssi INTEGER,
			hop_limit INTEGER,
			status TEXT DEFAULT 'received',
			reply_id INTEGER,
			error_reason TEXT
		);
		CREATE TABLE IF NOT EXISTS packets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			packet_id INTEGER,
			from_node INTEGER,
			to_node INTEGER,
			channel INTEGER,
			portnum INTEGER,
			timestamp INTEGER,
			rx_time INTEGER,
			rx_snr REAL,
			rx_rssi INTEGER,
			raw BLOB
		);
		CREATE INDEX IF NOT EXISTS idx_messages_channel ON messages(channel);
		CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);
		CREATE INDEX IF NOT EXISTS idx_packets_timestamp ON packets(timestamp);
		CREATE TABLE IF NOT EXISTS position_responses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			packet_id INTEGER,
			from_node INTEGER,
			requested_by INTEGER,
			latitude_i INTEGER,
			longitude_i INTEGER,
			altitude INTEGER,
			sats_in_view INTEGER,
			timestamp INTEGER
		);
		CREATE TABLE IF NOT EXISTS traceroute_responses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			packet_id INTEGER,
			from_node INTEGER,
			requested_by INTEGER,
			route TEXT,
			snr_towards TEXT,
			snr_back TEXT,
			hop_limit INTEGER,
			timestamp INTEGER
		);
		CREATE TABLE IF NOT EXISTS nodeinfo_responses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			packet_id INTEGER,
			from_node INTEGER,
			requested_by INTEGER,
			long_name TEXT,
			short_name TEXT,
			hw_model INTEGER,
			timestamp INTEGER
		);
		CREATE INDEX IF NOT EXISTS idx_position_responses_timestamp ON position_responses(timestamp);
		CREATE INDEX IF NOT EXISTS idx_traceroute_responses_timestamp ON traceroute_responses(timestamp);
		CREATE INDEX IF NOT EXISTS idx_nodeinfo_responses_timestamp ON nodeinfo_responses(timestamp)`,
		// v2: BBS mail table
		`CREATE TABLE IF NOT EXISTS mail (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			from_node INTEGER,
			to_node INTEGER,
			text TEXT,
			timestamp INTEGER,
			read INTEGER DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_mail_to_node ON mail(to_node);
		CREATE INDEX IF NOT EXISTS idx_mail_timestamp ON mail(timestamp)`,
	}

	for i, m := range migrations {
		v := i + 1
		if v <= version {
			continue
		}
		if _, err := d.db.Exec(m); err != nil {
			return fmt.Errorf("migration %d: %w", v, err)
		}
		d.db.Exec("INSERT INTO schema_version (version, applied_at) VALUES (?, ?)", v, time.Now().UnixMilli())
	}

	return nil
}
