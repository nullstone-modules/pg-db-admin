package postgresql

import (
	"database/sql"
	"fmt"
)

// Settings reports the server configuration that callers need to know before they attempt logical replication
//
// This is exposed over the api (`GET /settings`) so that infrastructure modules can detect a misconfigured
// instance up front instead of failing later with an opaque error from postgres or from a replication consumer.
type Settings struct {
	Version     string `json:"version"`
	CurrentUser string `json:"currentUser"`
	IsSuperuser bool   `json:"isSuperuser"`

	// WalLevel is `logical` when logical replication is available
	// On cloud sql, this is controlled by the `cloudsql.logical_decoding` database flag (requires an instance restart)
	WalLevel string `json:"walLevel"`

	MaxReplicationSlots  int `json:"maxReplicationSlots"`
	MaxWalSenders        int `json:"maxWalSenders"`
	UsedReplicationSlots int `json:"usedReplicationSlots"`
}

// LogicalReplicationEnabled reports whether the server can support logical replication slots
func (s Settings) LogicalReplicationEnabled() bool {
	return s.WalLevel == "logical"
}

type SettingsReader struct {
	DbOpener DbOpener
}

func (r *SettingsReader) Read() (*Settings, error) {
	db, err := r.DbOpener.OpenDatabase("")
	if err != nil {
		return nil, err
	}
	return ReadSettings(db)
}

func ReadSettings(db *sql.DB) (*Settings, error) {
	settings := &Settings{}

	sq := `SELECT
	version(),
	current_user,
	current_setting('wal_level'),
	current_setting('max_replication_slots')::int,
	current_setting('max_wal_senders')::int,
	(SELECT count(*) FROM pg_replication_slots)::int,
	(SELECT rolsuper FROM pg_roles WHERE rolname = current_user)`
	row := db.QueryRow(sq)
	err := row.Scan(&settings.Version, &settings.CurrentUser, &settings.WalLevel,
		&settings.MaxReplicationSlots, &settings.MaxWalSenders, &settings.UsedReplicationSlots, &settings.IsSuperuser)
	if err != nil {
		return nil, fmt.Errorf("error reading postgres settings: %w", err)
	}
	return settings, nil
}
