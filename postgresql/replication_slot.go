package postgresql

import (
	"database/sql"
	"fmt"
	"github.com/go-multierror/multierror"
	"github.com/nullstone-io/go-rest-api"
	"log"
	"time"
)

const (
	defaultReplicationSlotPlugin = "pgoutput"

	// dropSlotAttempts/dropSlotDelay bound how long Drop waits for a consumer to let go of a slot
	// The cloud function has a hard request timeout, so this stays well under it
	dropSlotAttempts = 10
	dropSlotDelay    = 500 * time.Millisecond
)

// ReplicationSlot defines a logical replication slot that a consumer (e.g. Datastream) reads changes from
//
// A slot is bound to a single database and retains WAL until its consumer has acknowledged it.
// An orphaned slot retains WAL forever, which eventually fills the instance disk -- Drop really drops the slot.
type ReplicationSlot struct {
	Name     string `json:"name"`
	Database string `json:"database"`
	// Plugin is the logical decoding output plugin, defaults to `pgoutput`
	Plugin string `json:"plugin"`

	// Active reports whether a consumer is currently streaming from this slot (read-only)
	Active bool `json:"active"`

	// Do not error if trying to create a slot that already exists
	// Instead, read the existing and return
	UseExisting bool `json:"useExisting"`
}

func (s ReplicationSlot) Key() ReplicationSlotKey {
	return ReplicationSlotKey{
		Database: s.Database,
		Name:     s.Name,
	}
}

type ReplicationSlotKey struct {
	Database string
	Name     string
}

var _ rest.DataAccess[ReplicationSlotKey, ReplicationSlot] = &ReplicationSlots{}

type ReplicationSlots struct {
	DbOpener DbOpener
}

func (r *ReplicationSlots) Create(obj ReplicationSlot) (*ReplicationSlot, error) {
	if obj.Plugin == "" {
		obj.Plugin = defaultReplicationSlotPlugin
	}

	if obj.UseExisting {
		if existing, err := r.Read(obj.Key()); err != nil {
			return nil, err
		} else if existing != nil {
			log.Printf("[Create] Replication slot %q already exists\n", obj.Name)
			if existing.Plugin != obj.Plugin {
				return nil, fmt.Errorf("replication slot %q already exists using the %q plugin, but is configured to use %q; "+
					"a slot's plugin cannot be changed in place", obj.Name, existing.Plugin, obj.Plugin)
			}
			return existing, nil
		}
	}

	// A logical slot is created in (and bound to) the database of the connection that creates it
	db, err := r.DbOpener.OpenDatabase(obj.Database)
	if err != nil {
		return nil, err
	}

	info, err := CalcDbConnectionInfo(db)
	if err != nil {
		return nil, fmt.Errorf("error analyzing database: %w", err)
	}
	if err := verifyLogicalDecoding(db, info); err != nil {
		return nil, err
	}

	log.Printf("Creating logical replication slot %q in database %q using plugin %q\n", obj.Name, obj.Database, obj.Plugin)
	sq := `SELECT pg_create_logical_replication_slot($1, $2)`
	if _, err := db.Exec(sq, obj.Name, obj.Plugin); err != nil {
		return nil, fmt.Errorf("error creating replication slot %q: %w", obj.Name, err)
	}
	return &obj, nil
}

func (r *ReplicationSlots) Read(key ReplicationSlotKey) (*ReplicationSlot, error) {
	db, err := r.DbOpener.OpenDatabase(key.Database)
	if err != nil {
		return nil, err
	}
	return r.read(db, key)
}

// Update cannot alter a slot -- postgres offers no way to change a slot's plugin or database
// It exists to ensure the slot is present and to report its current state
func (r *ReplicationSlots) Update(key ReplicationSlotKey, obj ReplicationSlot) (*ReplicationSlot, error) {
	obj.Name, obj.Database = key.Name, key.Database

	existing, err := r.Read(key)
	if err != nil {
		return nil, err
	} else if existing == nil {
		return r.Create(obj)
	}
	return existing, nil
}

// Drop really drops the slot
// Unlike the other resources in this api, leaving a logical replication slot behind is not harmless:
// the slot pins WAL indefinitely and will eventually fill the instance's disk
func (r *ReplicationSlots) Drop(key ReplicationSlotKey) (bool, error) {
	db, err := r.DbOpener.OpenDatabase(key.Database)
	if err != nil {
		return false, err
	}

	existing, err := r.read(db, key)
	if err != nil {
		return false, err
	} else if existing == nil {
		log.Printf("Replication slot %q does not exist, skipping drop\n", key.Name)
		return true, nil
	}

	var terminateErr error
	if existing.Active {
		// A slot cannot be dropped while a consumer is streaming from it
		// The consumer (e.g. a datastream stream) is expected to be torn down first, but it may not have disconnected yet
		log.Printf("Replication slot %q is still active, disconnecting its consumer\n", key.Name)
		if _, terminateErr = db.Exec(`SELECT pg_terminate_backend(active_pid) FROM pg_replication_slots WHERE slot_name = $1 AND active_pid IS NOT NULL`, key.Name); terminateErr != nil {
			// Terminating another role's backend requires membership in pg_signal_backend, which the admin user
			// does not necessarily have. The consumer may still disconnect on its own, so this is not fatal yet.
			log.Printf("Unable to disconnect the consumer of replication slot %q: %s\n", key.Name, terminateErr)
		}
	}

	var lastErr error
	for attempt := 0; attempt < dropSlotAttempts; attempt++ {
		log.Printf("Dropping replication slot %q\n", key.Name)
		if _, lastErr = db.Exec(`SELECT pg_drop_replication_slot($1)`, key.Name); lastErr == nil {
			return true, nil
		}
		// Releasing a slot is asynchronous -- the walsender clears active_pid shortly after it is terminated
		time.Sleep(dropSlotDelay)
	}

	errs := []error{fmt.Errorf("error dropping replication slot %q (a consumer is likely still attached to it -- "+
		"stop the consumer, then drop the slot): %w", key.Name, lastErr)}
	if terminateErr != nil {
		errs = append(errs, fmt.Errorf("error disconnecting its consumer: %w", terminateErr))
	}
	return false, multierror.New(errs)
}

func (r *ReplicationSlots) read(db *sql.DB, key ReplicationSlotKey) (*ReplicationSlot, error) {
	obj := ReplicationSlot{Name: key.Name}
	var slotType string
	sq := `SELECT slot_name, slot_type, coalesce(plugin, ''), coalesce(database, ''), active
FROM pg_replication_slots
WHERE slot_name = $1`
	row := db.QueryRow(sq, key.Name)
	if err := row.Scan(&obj.Name, &slotType, &obj.Plugin, &obj.Database, &obj.Active); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if slotType != "logical" {
		return nil, fmt.Errorf("replication slot %q already exists as a %s slot; refusing to manage it", key.Name, slotType)
	}
	if key.Database != "" && obj.Database != key.Database {
		return nil, fmt.Errorf("replication slot %q already exists in database %q, but is configured for database %q; "+
			"a slot's database cannot be changed -- use a different slot name", key.Name, obj.Database, key.Database)
	}
	return &obj, nil
}

// verifyLogicalDecoding produces an actionable error when the server cannot support logical slots
// Without it, postgres reports `logical decoding requires wal_level >= logical`, which says nothing about how to fix it
func verifyLogicalDecoding(db *sql.DB, info *DbInfo) error {
	if !info.SupportedFeatures.IsSupported(FeatureReplicationSlot) {
		return fmt.Errorf("logical replication slots require postgres 9.4 or later")
	}

	settings, err := ReadSettings(db)
	if err != nil {
		return err
	}
	if settings.WalLevel != "logical" {
		return fmt.Errorf("cannot create a logical replication slot: wal_level is %q, but logical replication requires %q "+
			"(on cloud sql, set the `cloudsql.logical_decoding` database flag to `on` -- this requires an instance restart)",
			settings.WalLevel, "logical")
	}
	return nil
}
