package acc

import (
	_ "github.com/lib/pq"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

// TestPublication verifies each publication shape and that membership is reconciled in place
// This requires the postgres server to run with wal_level=logical (see acc/docker-compose.yml)
func TestPublication(t *testing.T) {
	if os.Getenv("ACC") != "1" {
		t.Skip("Set ACC=1 to run e2e tests")
	}

	store := createStore(t)
	defer store.Close()

	db := openDb(t, "postgres")
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS publication_test_items ( id SERIAL NOT NULL, name varchar(255) )`)
	require.NoError(t, err, "create table")

	t.Run("for all tables", func(t *testing.T) {
		key := postgresql.PublicationKey{Database: "postgres", Name: "publication-test-all"}
		defer store.Publications.Drop(key)

		_, err := store.Publications.Create(postgresql.Publication{
			Name:      key.Name,
			Database:  key.Database,
			AllTables: true,
		})
		require.NoError(t, err, "create publication")

		find, err := store.Publications.Read(key)
		require.NoError(t, err, "read publication")
		require.NotNil(t, find, "publication not found")
		assert.True(t, find.AllTables, "expected `FOR ALL TABLES`")
	})

	t.Run("for tables in schema", func(t *testing.T) {
		key := postgresql.PublicationKey{Database: "postgres", Name: "publication-test-schema"}
		defer store.Publications.Drop(key)

		obj := postgresql.Publication{
			Name:        key.Name,
			Database:    key.Database,
			Schemas:     []string{"public"},
			UseExisting: true,
		}
		_, err := store.Publications.Create(obj)
		require.NoError(t, err, "create publication")

		find, err := store.Publications.Read(key)
		require.NoError(t, err, "read publication")
		require.NotNil(t, find, "publication not found")
		assert.False(t, find.AllTables, "expected a schema list, not `FOR ALL TABLES`")
		assert.Equal(t, []string{"public"}, find.Schemas)
		assert.Len(t, find.Tables, 0, "schema membership must not report individual tables")

		// Creating again adopts the existing publication instead of failing
		_, err = store.Publications.Create(obj)
		require.NoError(t, err, "re-create existing publication")
	})

	t.Run("reconciles membership without recreating", func(t *testing.T) {
		key := postgresql.PublicationKey{Database: "postgres", Name: "publication-test-reconcile"}
		defer store.Publications.Drop(key)

		_, err := store.Publications.Create(postgresql.Publication{
			Name:     key.Name,
			Database: key.Database,
			Schemas:  []string{"public"},
		})
		require.NoError(t, err, "create publication")

		// Swap whole-schema membership for a single explicit table
		_, err = store.Publications.Update(key, postgresql.Publication{
			Tables: []postgresql.PublicationTable{{Schema: "public", Name: "publication_test_items"}},
		})
		require.NoError(t, err, "update publication")

		find, err := store.Publications.Read(key)
		require.NoError(t, err, "read publication")
		require.NotNil(t, find, "publication not found")
		assert.Len(t, find.Schemas, 0, "expected the schema to be dropped from the publication")
		assert.Equal(t, []postgresql.PublicationTable{{Schema: "public", Name: "publication_test_items"}}, find.Tables)
	})

	t.Run("cannot convert to for all tables", func(t *testing.T) {
		key := postgresql.PublicationKey{Database: "postgres", Name: "publication-test-convert"}
		defer store.Publications.Drop(key)

		_, err := store.Publications.Create(postgresql.Publication{
			Name:     key.Name,
			Database: key.Database,
			Schemas:  []string{"public"},
		})
		require.NoError(t, err, "create publication")

		_, err = store.Publications.Update(key, postgresql.Publication{AllTables: true})
		require.Error(t, err, "expected converting to `FOR ALL TABLES` to be rejected")
		assert.Contains(t, err.Error(), "cannot alter this in place")
	})

	t.Run("drop", func(t *testing.T) {
		key := postgresql.PublicationKey{Database: "postgres", Name: "publication-test-drop"}

		_, err := store.Publications.Create(postgresql.Publication{
			Name:     key.Name,
			Database: key.Database,
			Schemas:  []string{"public"},
		})
		require.NoError(t, err, "create publication")

		dropped, err := store.Publications.Drop(key)
		require.NoError(t, err, "drop publication")
		assert.True(t, dropped)

		find, err := store.Publications.Read(key)
		require.NoError(t, err, "read publication")
		assert.Nil(t, find, "expected the publication to be gone")

		// Dropping a publication that no longer exists is not an error
		_, err = store.Publications.Drop(key)
		require.NoError(t, err, "drop missing publication")
	})
}
