package acc

import (
	_ "github.com/lib/pq"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

// TestTablePrivilege verifies read-only access to a schema, including tables created after the grant
func TestTablePrivilege(t *testing.T) {
	if os.Getenv("ACC") != "1" {
		t.Skip("Set ACC=1 to run e2e tests")
	}

	store := createStore(t)
	defer store.Close()

	reader := "table-privilege-test-reader"
	db := openDb(t, "postgres")
	dropRole(t, db, reader)

	_, err := store.Roles.Create(postgresql.Role{Name: reader, Password: "reader-password", UseExisting: true})
	require.NoError(t, err, "create reader role")

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS table_privilege_test_existing ( id SERIAL NOT NULL )`)
	require.NoError(t, err, "create existing table")

	hasSelect := func(t *testing.T, table string) bool {
		var granted bool
		require.NoError(t, db.QueryRow(`SELECT has_table_privilege($1, $2, 'SELECT')`, reader, table).Scan(&granted))
		return granted
	}

	require.False(t, hasSelect(t, "table_privilege_test_existing"), "reader should not have access before the grant")

	_, err = store.TablePrivileges.Create(postgresql.TablePrivilege{
		Database:              "postgres",
		Schema:                "public",
		Role:                  reader,
		IncludeFuture:         true,
		FutureFromTableOwners: true,
		GrantConnect:          true,
	})
	require.NoError(t, err, "grant table privileges")

	assert.True(t, hasSelect(t, "table_privilege_test_existing"), "reader should have select on existing tables")

	t.Run("covers tables created later by the same owners", func(t *testing.T) {
		// The grant discovered the owners of the schema's tables, so tables they create later are covered too
		_, err := db.Exec(`CREATE TABLE IF NOT EXISTS table_privilege_test_future ( id SERIAL NOT NULL )`)
		require.NoError(t, err, "create future table")

		assert.True(t, hasSelect(t, "table_privilege_test_future"), "reader should have select on tables created after the grant")
	})

	t.Run("re-applying is idempotent", func(t *testing.T) {
		_, err := store.TablePrivileges.Create(postgresql.TablePrivilege{
			Database:              "postgres",
			Schema:                "public",
			Role:                  reader,
			IncludeFuture:         true,
			FutureFromTableOwners: true,
		})
		require.NoError(t, err, "re-grant table privileges")
		assert.True(t, hasSelect(t, "table_privilege_test_existing"))
	})

	t.Run("every schema", func(t *testing.T) {
		_, err := db.Exec(`CREATE SCHEMA IF NOT EXISTS table_privilege_test_schema`)
		require.NoError(t, err, "create schema")
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS table_privilege_test_schema.widgets ( id SERIAL NOT NULL )`)
		require.NoError(t, err, "create table in schema")

		_, err = store.TablePrivileges.Create(postgresql.TablePrivilege{
			Database:      "postgres",
			Schema:        postgresql.AllSchemas,
			Role:          reader,
			IncludeFuture: true,
		})
		require.NoError(t, err, "grant table privileges on every schema")

		assert.True(t, hasSelect(t, "table_privilege_test_schema.widgets"))
	})

	t.Run("rejects an unknown privilege", func(t *testing.T) {
		_, err := store.TablePrivileges.Create(postgresql.TablePrivilege{
			Database:   "postgres",
			Schema:     "public",
			Role:       reader,
			Privileges: []string{"SELECT; DROP TABLE todos"},
		})
		require.Error(t, err, "expected an invalid privilege to be rejected")
		assert.Contains(t, err.Error(), "invalid table privilege")
	})
}
