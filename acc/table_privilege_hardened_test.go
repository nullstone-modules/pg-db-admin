package acc

import (
	_ "github.com/lib/pq"
	"github.com/nullstone-modules/pg-db-admin/postgresql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

// TestTablePrivilegeInHardenedDatabase covers a database that does not rely on PUBLIC's default privileges.
//
// `GRANT CONNECT ON DATABASE` and `GRANT USAGE ON SCHEMA` are only a warning when the grantor cannot grant
// them -- the statement succeeds having granted nothing. Where PUBLIC still holds the defaults that goes
// unnoticed, because the role can connect anyway. Where those defaults were revoked, the role silently ends up
// unable to read anything, and the failure surfaces much later in the replication consumer.
func TestTablePrivilegeInHardenedDatabase(t *testing.T) {
	if os.Getenv("ACC") != "1" {
		t.Skip("Set ACC=1 to run e2e tests")
	}

	const (
		admin        = "tp-hardened-admin"
		owner        = "tp-hardened-owner"
		reader       = "tp-hardened-reader"
		password     = "tp-hardened-password"
		databaseName = "tp_hardened_test"
	)

	superDb := openDb(t, "postgres")
	_, err := superDb.Exec(`DROP DATABASE IF EXISTS ` + databaseName + ` WITH (FORCE)`)
	require.NoError(t, err, "reset database")
	dropRole(t, superDb, reader)
	dropRole(t, superDb, owner)
	dropRole(t, superDb, admin)

	_, err = superDb.Exec(`CREATE ROLE "tp-hardened-admin" LOGIN PASSWORD '` + password + `' CREATEROLE`)
	require.NoError(t, err, "create admin role")

	adminStore := createStoreAs(t, admin, password)
	defer adminStore.Close()

	_, err = adminStore.Roles.Create(postgresql.Role{Name: owner, Password: password, UseExisting: true})
	require.NoError(t, err, "create owner role")
	_, err = adminStore.Roles.Create(postgresql.Role{Name: reader, Password: password, UseExisting: true})
	require.NoError(t, err, "create reader role")

	_, err = superDb.Exec(`CREATE DATABASE ` + databaseName + ` OWNER "tp-hardened-owner"`)
	require.NoError(t, err, "create database owned by the owner role")

	// Take away the defaults every role would otherwise inherit through PUBLIC, but leave the admin able to
	// connect -- it has to reach the database to administer it at all
	_, err = superDb.Exec(`REVOKE CONNECT ON DATABASE ` + databaseName + ` FROM PUBLIC`)
	require.NoError(t, err, "revoke connect from public")
	_, err = superDb.Exec(`GRANT CONNECT ON DATABASE ` + databaseName + ` TO "tp-hardened-admin"`)
	require.NoError(t, err, "let the admin connect")

	appDb := openDb(t, databaseName)
	defer appDb.Close()
	_, err = appDb.Exec(`REVOKE USAGE ON SCHEMA public FROM PUBLIC`)
	require.NoError(t, err, "revoke schema usage from public")

	ownerDb := openDbAs(t, owner, password, databaseName)
	defer ownerDb.Close()
	_, err = ownerDb.Exec(`CREATE TABLE items ( id SERIAL NOT NULL )`)
	require.NoError(t, err, "create table owned by the owner role")

	canConnect := func(t *testing.T) bool {
		var granted bool
		require.NoError(t, appDb.QueryRow(`SELECT has_database_privilege($1, $2, 'CONNECT')`, reader, databaseName).Scan(&granted))
		return granted
	}
	canUseSchema := func(t *testing.T) bool {
		var granted bool
		require.NoError(t, appDb.QueryRow(`SELECT has_schema_privilege($1, 'public', 'USAGE')`, reader).Scan(&granted))
		return granted
	}
	require.False(t, canConnect(t), "reader should not be able to connect before the grant")
	require.False(t, canUseSchema(t), "reader should not have schema access before the grant")

	_, err = adminStore.TablePrivileges.Create(postgresql.TablePrivilege{
		Database:              databaseName,
		Schema:                "public",
		Role:                  reader,
		IncludeFuture:         true,
		FutureFromTableOwners: true,
		GrantConnect:          true,
	})
	require.NoError(t, err, "grant table privileges in a hardened database")

	assert.True(t, canConnect(t), "reader should be able to connect")
	assert.True(t, canUseSchema(t), "reader should have schema access")

	var canSelect bool
	require.NoError(t, appDb.QueryRow(`SELECT has_table_privilege($1, 'items', 'SELECT')`, reader).Scan(&canSelect))
	assert.True(t, canSelect, "reader should have select on the owner's table")
}
