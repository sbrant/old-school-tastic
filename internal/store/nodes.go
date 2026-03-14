package store

import (
	"database/sql"
	"time"
)

type Node struct {
	Num                int64
	UserID             sql.NullString
	LongName           sql.NullString
	ShortName          sql.NullString
	HwModel            sql.NullInt64
	Role               sql.NullInt64
	LatitudeI          sql.NullInt64
	LongitudeI         sql.NullInt64
	Altitude           sql.NullInt64
	SNR                sql.NullFloat64
	LastHeard          sql.NullInt64
	BatteryLevel       sql.NullInt64
	Voltage            sql.NullFloat64
	ChannelUtilization sql.NullFloat64
	AirUtilTx          sql.NullFloat64
	Channel            sql.NullInt64
	ViaMqtt            sql.NullBool
	HopsAway           sql.NullInt64
	IsFavorite         sql.NullBool
	PublicKey          []byte
}

func (d *DB) UpsertNode(n Node) error {
	_, err := d.db.Exec(`
		INSERT INTO nodes (num, user_id, long_name, short_name, hw_model, role,
			latitude_i, longitude_i, altitude, snr, last_heard,
			battery_level, voltage, channel_utilization, air_util_tx,
			channel, via_mqtt, hops_away, is_favorite, public_key, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(num) DO UPDATE SET
			user_id = COALESCE(excluded.user_id, user_id),
			long_name = COALESCE(excluded.long_name, long_name),
			short_name = COALESCE(excluded.short_name, short_name),
			hw_model = COALESCE(excluded.hw_model, hw_model),
			role = COALESCE(excluded.role, role),
			latitude_i = COALESCE(excluded.latitude_i, latitude_i),
			longitude_i = COALESCE(excluded.longitude_i, longitude_i),
			altitude = COALESCE(excluded.altitude, altitude),
			snr = COALESCE(excluded.snr, snr),
			last_heard = COALESCE(excluded.last_heard, last_heard),
			battery_level = COALESCE(excluded.battery_level, battery_level),
			voltage = COALESCE(excluded.voltage, voltage),
			channel_utilization = COALESCE(excluded.channel_utilization, channel_utilization),
			air_util_tx = COALESCE(excluded.air_util_tx, air_util_tx),
			channel = COALESCE(excluded.channel, channel),
			via_mqtt = COALESCE(excluded.via_mqtt, via_mqtt),
			hops_away = COALESCE(excluded.hops_away, hops_away),
			is_favorite = COALESCE(excluded.is_favorite, is_favorite),
			public_key = COALESCE(excluded.public_key, public_key),
			updated_at = excluded.updated_at`,
		n.Num, n.UserID, n.LongName, n.ShortName, n.HwModel, n.Role,
		n.LatitudeI, n.LongitudeI, n.Altitude, n.SNR, n.LastHeard,
		n.BatteryLevel, n.Voltage, n.ChannelUtilization, n.AirUtilTx,
		n.Channel, n.ViaMqtt, n.HopsAway, n.IsFavorite, n.PublicKey,
		time.Now().UnixMilli())
	return err
}

func (d *DB) GetAllNodes() ([]Node, error) {
	rows, err := d.db.Query("SELECT num, user_id, long_name, short_name, hw_model, role, latitude_i, longitude_i, altitude, snr, last_heard, battery_level, voltage, channel_utilization, air_util_tx, channel, via_mqtt, hops_away, is_favorite, public_key FROM nodes ORDER BY hops_away ASC, last_heard DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		var n Node
		err := rows.Scan(&n.Num, &n.UserID, &n.LongName, &n.ShortName, &n.HwModel, &n.Role,
			&n.LatitudeI, &n.LongitudeI, &n.Altitude, &n.SNR, &n.LastHeard,
			&n.BatteryLevel, &n.Voltage, &n.ChannelUtilization, &n.AirUtilTx,
			&n.Channel, &n.ViaMqtt, &n.HopsAway, &n.IsFavorite, &n.PublicKey)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, nil
}

func (d *DB) GetNodeName(num uint32) string {
	var short, long sql.NullString
	d.db.QueryRow("SELECT short_name, long_name FROM nodes WHERE num = ?", num).Scan(&short, &long)
	if short.Valid && short.String != "" {
		return short.String
	}
	if long.Valid && long.String != "" {
		return long.String
	}
	return ""
}
