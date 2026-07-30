package acc

import (
	_ "github.com/lib/pq"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

// TestReplicationSlot verifies that a logical slot is created, adopted, and -- importantly -- really dropped
// An orphaned slot pins WAL forever, so Drop must not be a no-op
// This requires the postgres server to run with wal_level=logical (see acc/docker-compose.yml)
func TestReplicationSlot(t *testing.T) {
	if os.Getenv("ACC") != "1" {
		t.Skip("Set ACC=1 to run e2e tests")
	}

	store := createStore(t)
	defer store.Close()

	key := postgresql.ReplicationSlotKey{Database: "postgres", Name: "replication_slot_test"}
	defer store.ReplicationSlots.Drop(key)

	created, err := store.ReplicationSlots.Create(postgresql.ReplicationSlot{
		Name:     key.Name,
		Database: key.Database,
	})
	require.NoError(t, err, "create replication slot")
	assert.Equal(t, "pgoutput", created.Plugin, "expected the default plugin")

	find, err := store.ReplicationSlots.Read(key)
	require.NoError(t, err, "read replication slot")
	require.NotNil(t, find, "replication slot not found")
	assert.Equal(t, "pgoutput", find.Plugin)
	assert.Equal(t, "postgres", find.Database)
	assert.False(t, find.Active, "expected no consumer attached")

	t.Run("adopts an existing slot", func(t *testing.T) {
		_, err := store.ReplicationSlots.Create(postgresql.ReplicationSlot{
			Name:        key.Name,
			Database:    key.Database,
			UseExisting: true,
		})
		require.NoError(t, err, "re-create existing replication slot")
	})

	t.Run("rejects a plugin change", func(t *testing.T) {
		_, err := store.ReplicationSlots.Create(postgresql.ReplicationSlot{
			Name:        key.Name,
			Database:    key.Database,
			Plugin:      "test_decoding",
			UseExisting: true,
		})
		require.Error(t, err, "expected a plugin change to be rejected")
		assert.Contains(t, err.Error(), "cannot be changed in place")
	})

	t.Run("drop", func(t *testing.T) {
		dropped, err := store.ReplicationSlots.Drop(key)
		require.NoError(t, err, "drop replication slot")
		assert.True(t, dropped)

		find, err := store.ReplicationSlots.Read(key)
		require.NoError(t, err, "read replication slot")
		assert.Nil(t, find, "expected the replication slot to be dropped")

		// Dropping a slot that no longer exists is not an error
		_, err = store.ReplicationSlots.Drop(key)
		require.NoError(t, err, "drop missing replication slot")
	})
}
